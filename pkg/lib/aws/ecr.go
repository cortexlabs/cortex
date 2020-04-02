package aws

import (
	"context"
	"strings"
	"encoding/base64"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type ECRAuthConfig struct {
	Username string
	AccessToken string
	ProxyEndpoint string
}

func GetECRAuthToken() (*ecr.GetAuthorizationTokenOutput, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := ecr.New(sess)
	input := &ecr.GetAuthorizationTokenInput{}
	result, err := svc.GetAuthorizationTokenWithContext(context.Background(), input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				return result, errors.Wrap(aerr, ecr.ErrCodeServerException, "failed to retrieve ECR auth token")
			case ecr.ErrCodeInvalidParameterException:
				return result, errors.Wrap(
					aerr, 
					ecr.ErrCodeInvalidParameterException, 
					"failed to retrieve ECR auth token",
				)
			default:
				return result, errors.Wrap(aerr, "failed to retrieve ECR auth token")
			}
		} else {
			return result, errors.Wrap(err, "failed to retrieve ECR auth token")
		}
	}

	return result, nil
}

func ExtractECRAuthConfigFromTokenOutput(auth *ecr.GetAuthorizationTokenOutput) (ECRAuthConfig, error) {
	var authConfig ECRAuthConfig
	if len(auth.AuthorizationData) < 1 {
		return authConfig, ErrorInvalidAuthorizationTokenOutput(
			"GetAuthorizationTokenOutput.AuthorizationData field is empty",
		)
	}
	authData := auth.AuthorizationData[0]

	credentials, err := base64.URLEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		return authConfig, ErrorInvalidAuthorizationTokenOutput(
			"GetAuthorizationTokenOutput can't be decoded with base64",
		)
	}
	credentialsString := string(credentials)
	splitCredentials := strings.Split(credentialsString, ":")
	if len(splitCredentials) != 2 {
		return authConfig, ErrorInvalidAuthorizationTokenOutput(
			"GetAuthorizationTokenOutput doesn't have user:token encoded",
		)
	}

	authConfig.Username = splitCredentials[0]
	authConfig.AccessToken = splitCredentials[1]
	authConfig.ProxyEndpoint = *authData.ProxyEndpoint

	return authConfig, nil
} 