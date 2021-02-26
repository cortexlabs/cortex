/*
Copyright 2021 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aws

import (
	"encoding/base64"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/private/protocol/query"
	"github.com/aws/aws-sdk-go/private/protocol/xml/xmlutil"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	libjson "github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
)

// Returns account ID, whether the credentials were valid, any other error that occurred
// Ignores cache, so will re-run on every call to this method
func (c *Client) CheckCredentials() (string, string, error) {
	response, err := c.STS().GetCallerIdentity(nil)
	if err != nil {
		return "", "", ErrorInvalidAWSCredentials(err)
	}

	c.accountID = response.Account
	c.hashedAccountID = pointer.String(hash.String(*c.accountID))

	return *c.accountID, *c.hashedAccountID, nil
}

// Only re-checks the credentials if they have never been checked (so will not catch e.g. credentials expiring or getting revoked)
func (c *Client) GetCachedAccountID() (string, string, error) {
	if c.accountID == nil || c.hashedAccountID == nil {
		if _, _, err := c.CheckCredentials(); err != nil {
			return "", "", err
		}
	}
	return *c.accountID, *c.hashedAccountID, nil
}

type awsRequest struct {
	Header        http.Header
	URL           string
	Method        string
	Host          string
	Body          string
	ContentLength int64
}

func (c *Client) IdentityRequestAsHeader() (string, error) {
	req, _ := c.STS().GetCallerIdentityRequest(nil)

	err := req.Sign()
	if err != nil {
		return "", errors.WithStack(err)
	}

	reqBody, err := ioutil.ReadAll(req.HTTPRequest.Body)
	if err != nil {
		return "", errors.WithStack(err)
	}

	signedRequestArtifacts := awsRequest{
		Header:        req.HTTPRequest.Header,
		URL:           req.HTTPRequest.URL.String(),
		Method:        req.HTTPRequest.Method,
		Host:          req.HTTPRequest.Host,
		Body:          string(reqBody),
		ContentLength: req.HTTPRequest.ContentLength,
	}
	jsonSignedRequestArtifacts, err := libjson.Marshal(signedRequestArtifacts)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(jsonSignedRequestArtifacts), nil
}

// ExecuteIdentityRequestFromHeader executes identity request marshalled from header and returns account id if successful
func ExecuteIdentityRequestFromHeader(indentityRequestheader string) (string, error) {
	jsonObj, err := base64.RawURLEncoding.DecodeString(indentityRequestheader)
	if err != nil {
		return "", errors.WithStack(err)
	}

	signedRequestArtifacts := awsRequest{}
	err = libjson.Unmarshal(jsonObj, &signedRequestArtifacts)
	if err != nil {
		return "", err
	}

	httpClient := http.Client{}

	url, err := url.Parse(signedRequestArtifacts.URL)
	if err != nil {
		return "", errors.WithStack(err)
	}

	req := http.Request{
		Header:        signedRequestArtifacts.Header,
		Method:        signedRequestArtifacts.Method,
		URL:           url,
		Body:          ioutil.NopCloser(strings.NewReader(signedRequestArtifacts.Body)),
		ContentLength: signedRequestArtifacts.ContentLength,
		Host:          signedRequestArtifacts.Host,
	}

	resp, err := httpClient.Do(&req)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		awsReq := request.Request{HTTPResponse: resp}
		query.UnmarshalError(&awsReq)
		return "", errors.WithStack(awsReq.Error)
	}

	decoder := xml.NewDecoder(resp.Body)

	result := sts.GetCallerIdentityOutput{}
	err = xmlutil.UnmarshalXML(&result, decoder, "GetCallerIdentityResult")
	if err != nil {
		return "", awserr.NewRequestFailure(
			awserr.New(request.ErrCodeSerialization, "failed decoding Query response", err),
			resp.StatusCode,
			resp.Header.Get("X-Amzn-Requestid"),
		)
	}
	if result.Account == nil {
		return "", errors.ErrorUnexpected("GetCallerIdentityResult xml parsing failed")
	}

	return *result.Account, nil
}
