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
	"strings"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

func (c *Client) GetUser() (iam.User, error) {
	getUserOutput, err := c.IAM().GetUser(nil)
	if err != nil {
		return iam.User{}, errors.WithStack(err)
	}
	return *getUserOutput.User, nil
}

func (c *Client) GetGroupsForUser(userName string) ([]iam.Group, error) {
	input := &iam.ListGroupsForUserInput{
		UserName: &userName,
	}

	var groups []iam.Group

	err := c.IAM().ListGroupsForUserPages(input, func(page *iam.ListGroupsForUserOutput, lastPage bool) bool {
		for _, group := range page.Groups {
			groups = append(groups, *group)
		}
		return true
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return groups, nil
}

// Note: root users don't have attached policies, but do have full access
func (c *Client) GetManagedPoliciesForUser(userName string) ([]iam.AttachedPolicy, error) {
	var policies []iam.AttachedPolicy

	userManagedPolicies, err := c.IAM().ListAttachedUserPolicies(&iam.ListAttachedUserPoliciesInput{
		UserName: &userName,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for _, policy := range userManagedPolicies.AttachedPolicies {
		policies = append(policies, *policy)
	}

	groups, err := c.GetGroupsForUser(userName)
	if err != nil {
		return nil, err
	}

	for _, group := range groups {
		groupManagedPolicies, err := c.IAM().ListAttachedGroupPolicies(&iam.ListAttachedGroupPoliciesInput{
			GroupName: group.GroupName,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}
		for _, policy := range groupManagedPolicies.AttachedPolicies {
			policies = append(policies, *policy)
		}
	}

	return policies, nil
}

func (c *Client) IsAdmin() bool {
	user, err := c.GetUser()
	if err != nil {
		return false
	}

	// Root users may not have a user name
	if user.UserName == nil {
		return true
	}

	// Root users may have a user name
	if user.Arn == nil || strings.HasSuffix(*user.Arn, ":root") {
		return true
	}

	policies, err := c.GetManagedPoliciesForUser(*user.UserName)
	if err != nil {
		return false
	}

	for _, policy := range policies {
		if *policy.PolicyArn == "arn:aws:iam::aws:policy/AdministratorAccess" {
			return true
		}
	}

	return false
}
