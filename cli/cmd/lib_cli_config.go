/*
Copyright 2020 Cortex Labs, Inc.

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

package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	cr "github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/files"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/yaml"
)

type CLIConfig struct {
	Telemetry bool       `json:"telemetry" yaml:"telemetry"`
	Profiles  []*Profile `json:"profiles" yaml:"profiles"`
}

type Profile struct {
	Name               string   `json:"name" yaml:"name"`
	Provider           Provider `json:"provider" yaml:"provider"`
	OperatorEndpoint   *string  `json:"operator_endpoint,omitempty" yaml:"operator_endpoint,omitempty"`
	AWSAccessKeyID     *string  `json:"aws_access_key_id,omitempty" yaml:"aws_access_key_id,omitempty"`
	AWSSecretAccessKey *string  `json:"aws_secret_access_key,omitempty" yaml:"aws_secret_access_key,omitempty"`
}

var _cliConfigValidation = &cr.StructValidation{
	TreatNullAsEmpty: true,
	StructFieldValidations: []*cr.StructFieldValidation{
		{
			StructField: "Telemetry",
			BoolValidation: &cr.BoolValidation{
				Default:  true,
				Required: false,
			},
		},
		{
			StructField: "Profiles",
			StructListValidation: &cr.StructListValidation{
				AllowExplicitNull: true,
				StructValidation: &cr.StructValidation{
					StructFieldValidations: []*cr.StructFieldValidation{
						{
							StructField: "Name",
							StringValidation: &cr.StringValidation{
								Required:  true,
								MaxLength: 63,
							},
						},
						{
							StructField: "Provider",
							StringValidation: &cr.StringValidation{
								Required:      true,
								AllowedValues: ProviderStrings(),
							},
							Parser: func(str string) (interface{}, error) {
								return ProviderFromString(str), nil
							},
						},
						{
							StructField: "OperatorEndpoint",
							StringPtrValidation: &cr.StringPtrValidation{
								Required:  false,
								Validator: cr.GetURLValidator(false, false),
							},
						},
						{
							StructField: "AWSAccessKeyID",
							StringPtrValidation: &cr.StringPtrValidation{
								Required: false,
							},
						},
						{
							StructField: "AWSSecretAccessKey",
							StringPtrValidation: &cr.StringPtrValidation{
								Required: false,
							},
						},
					},
				},
			},
		},
	},
}

func (cliConfig *CLIConfig) validate() error {
	profileNames := strset.New()

	for _, profile := range cliConfig.Profiles {
		if profileNames.Has(profile.Name) {
			return errors.Wrap(ErrorDuplicateProfileNames(profile.Name), "profiles")
		}

		profileNames.Add(profile.Name)

		if err := profile.validate(); err != nil {
			return errors.Wrap(err, "profiles")
		}
	}

	// Ensure the local profile is always present
	if !profileNames.Has(Local.String()) {
		localProfile := &Profile{
			Name:     Local.String(),
			Provider: Local,
		}

		cliConfig.Profiles = append([]*Profile{localProfile}, cliConfig.Profiles...)
	}

	return nil
}

func (profile *Profile) validate() error {
	if profile.Name == "" {
		return errors.Wrap(cr.ErrorMustBeDefined(), "name")
	}
	if profile.Provider == UnknownProvider {
		return errors.Wrap(cr.ErrorMustBeDefined(ProviderStrings()), profile.Name, "provider")
	}

	if err := checkReservedProfileNames(profile.Name, profile.Provider); err != nil {
		return err
	}

	if profile.Provider == Local {
		if profile.OperatorEndpoint != nil {
			return errors.Wrap(ErrorOperatorEndpointInLocalProfile(), profile.Name)
		}
	}

	if profile.Provider == AWS {
		if profile.OperatorEndpoint == nil {
			return errors.Wrap(cr.ErrorMustBeDefined(), profile.Name, "operator_endpoint")
		}
		if profile.AWSAccessKeyID == nil {
			return errors.Wrap(cr.ErrorMustBeDefined(), profile.Name, "aws_access_key_id")
		}
		if profile.AWSSecretAccessKey == nil {
			return errors.Wrap(cr.ErrorMustBeDefined(), profile.Name, "aws_secret_access_key")
		}
	}

	return nil
}

func checkReservedProfileNames(profileName string, provider Provider) error {
	profileNameProvider := ProviderFromString(profileName)
	if profileNameProvider == UnknownProvider {
		return nil
	}

	if profileNameProvider != provider {
		return ErrorProfileProviderNameConflict(profileName, provider)
	}

	return nil
}

func providerPromptValidation(profileName string, defaults Profile) *cr.PromptValidation {
	defaultProviderStr := ""
	if defaults.Provider != UnknownProvider {
		defaultProviderStr = defaults.Provider.String()
	}

	return &cr.PromptValidation{
		SkipNonEmptyFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "Provider",
				PromptOpts: &prompt.Options{
					Prompt: fmt.Sprintf("provider (%s)", s.StrsOr(ProviderStrings())),
				},
				StringValidation: &cr.StringValidation{
					Required:      true,
					Default:       defaultProviderStr,
					AllowedValues: ProviderStrings(),
				},
				Parser: func(str string) (interface{}, error) {
					provider := ProviderFromString(str)
					if err := checkReservedProfileNames(profileName, provider); err != nil {
						return nil, err
					}
					return provider, nil
				},
			},
		},
	}
}

func localProfilePromptValidation(defaults Profile) *cr.PromptValidation {
	accessKeyIDPrompt := "aws access key id"
	if defaults.AWSAccessKeyID == nil {
		accessKeyIDPrompt += " [press ENTER to skip]"
	}

	secretAccessKeyPrompt := "aws secret access key"
	if defaults.AWSSecretAccessKey == nil {
		secretAccessKeyPrompt += " [press ENTER to skip]"
	}

	return &cr.PromptValidation{
		SkipNonEmptyFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "AWSAccessKeyID",
				PromptOpts: &prompt.Options{
					Prompt: accessKeyIDPrompt,
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required:   false,
					AllowEmpty: true,
					Default:    defaults.AWSAccessKeyID,
				},
			},
			{
				StructField: "AWSSecretAccessKey",
				PromptOpts: &prompt.Options{
					Prompt:      secretAccessKeyPrompt,
					MaskDefault: true,
					HideTyping:  true,
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required:   false,
					AllowEmpty: true,
					Default:    defaults.AWSSecretAccessKey,
				},
			},
		},
	}
}

func awsProfilePromptValidation(defaults Profile) *cr.PromptValidation {
	return &cr.PromptValidation{
		SkipNonEmptyFields: true,
		PromptItemValidations: []*cr.PromptItemValidation{
			{
				StructField: "OperatorEndpoint",
				PromptOpts: &prompt.Options{
					Prompt: "cortex operator endpoint",
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required:  true,
					Default:   defaults.OperatorEndpoint,
					Validator: validateOperatorEndpoint,
				},
			},
			{
				StructField: "AWSAccessKeyID",
				PromptOpts: &prompt.Options{
					Prompt: "aws access key id",
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required: true,
					Default:  defaults.AWSAccessKeyID,
				},
			},
			{
				StructField: "AWSSecretAccessKey",
				PromptOpts: &prompt.Options{
					Prompt:      "aws secret access key",
					MaskDefault: true,
					HideTyping:  true,
				},
				StringPtrValidation: &cr.StringPtrValidation{
					Required: true,
					Default:  defaults.AWSSecretAccessKey,
				},
			},
		},
	}
}

// Only validate this during prompt, not when reading from file
func validateOperatorEndpoint(endpoint string) (string, error) {
	url, err := cr.GetURLValidator(false, false)(endpoint)
	if err != nil {
		return "", err
	}

	parsedURL, err := urls.Parse(url)
	if err != nil {
		return "", err
	}

	parsedURL.Scheme = "https"

	url = parsedURL.String()

	req, err := http.NewRequest("GET", urls.Join(url, "/verifycortex"), nil)
	if err != nil {
		return "", errors.Wrap(err, "verifying operator endpoint", url)
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	response, err := client.Do(req)
	if err != nil {
		return "", ErrorInvalidOperatorEndpoint(url)
	}

	if response.StatusCode != 200 {
		return "", ErrorInvalidOperatorEndpoint(url)
	}

	return url, nil
}

func readTelemetryConfig() (bool, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return false, err
	}

	return cliConfig.Telemetry, nil
}

// Returns false if there is an error reading the CLI config
func isTelemetryEnabled() bool {
	enabled, err := readTelemetryConfig()
	if err != nil {
		return false
	}
	return enabled
}

// Will return nil if not configured, except for local
func readProfile(profileName string) (*Profile, error) {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return nil, err
	}

	for _, profile := range cliConfig.Profiles {
		if profile.Name == profileName {
			return profile, nil
		}
	}

	if profileName == Local.String() {
		return &Profile{
			Name:     Local.String(),
			Provider: Local,
		}, nil
	}

	return nil, nil
}

func readOrConfigureNonLocalProfile(profileName string) (Profile, error) {
	profile, err := readProfile(profileName)
	if err != nil {
		return Profile{}, err
	}

	if profile != nil {
		if profile.Provider == Local {
			return Profile{}, ErrorLocalProviderNotSupported(*profile)
		}
		return *profile, nil
	}

	fieldsToSkipPrompt := Profile{
		Provider: AWS,
	}
	return configureProfile(profileName, fieldsToSkipPrompt)
}

func getDefaultProfileConfig(profileName string) Profile {
	defaults := Profile{}

	prevProfile, err := readProfile(profileName)
	if err == nil && prevProfile != nil {
		defaults = *prevProfile
	}

	if defaults.Provider == UnknownProvider {
		if profileNameProvider := ProviderFromString(profileName); profileNameProvider != UnknownProvider {
			defaults.Provider = profileNameProvider
		}
	}

	if defaults.AWSAccessKeyID == nil && os.Getenv("AWS_ACCESS_KEY_ID") != "" {
		defaults.AWSAccessKeyID = pointer.String(os.Getenv("AWS_ACCESS_KEY_ID"))
	}
	if defaults.AWSSecretAccessKey == nil && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
		defaults.AWSSecretAccessKey = pointer.String(os.Getenv("AWS_SECRET_ACCESS_KEY"))
	}
	if defaults.OperatorEndpoint == nil && os.Getenv("CORTEX_OPERATOR_ENDPOINT") != "" {
		defaults.OperatorEndpoint = pointer.String(os.Getenv("CORTEX_OPERATOR_ENDPOINT"))
	}

	return defaults
}

func configureProfile(profileName string, fieldsToSkipPrompt Profile) (Profile, error) {
	fmt.Println("profile: " + profileName + "\n")

	defaults := getDefaultProfileConfig(profileName)

	if fieldsToSkipPrompt.Provider == UnknownProvider {
		if profileNameProvider := ProviderFromString(profileName); profileNameProvider != UnknownProvider {
			fieldsToSkipPrompt.Provider = profileNameProvider
		}
	}

	profile := Profile{
		Name:               profileName,
		Provider:           fieldsToSkipPrompt.Provider,
		OperatorEndpoint:   fieldsToSkipPrompt.OperatorEndpoint,
		AWSAccessKeyID:     fieldsToSkipPrompt.AWSAccessKeyID,
		AWSSecretAccessKey: fieldsToSkipPrompt.AWSSecretAccessKey,
	}

	err := cr.ReadPrompt(&profile, providerPromptValidation(profileName, defaults))
	if err != nil {
		return Profile{}, err
	}

	switch profile.Provider {
	case Local:
		err = cr.ReadPrompt(&profile, localProfilePromptValidation(defaults))
	case AWS:
		err = cr.ReadPrompt(&profile, awsProfilePromptValidation(defaults))
	}
	if err != nil {
		return Profile{}, err
	}

	if err := profile.validate(); err != nil {
		return Profile{}, err
	}

	if err := addProfileToCLIConfig(profile); err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func readCLIConfig() (CLIConfig, error) {
	if !files.IsFile(_cliConfigPath) {
		// add empty file so that the file created by the manager container maintains current user permissions
		files.MakeEmptyFile(_cliConfigPath)

		return CLIConfig{
			Telemetry: true,
		}, nil
	}

	cliConfig := CLIConfig{}
	errs := cr.ParseYAMLFile(&cliConfig, _cliConfigValidation, _cliConfigPath)
	if errors.HasError(errs) {
		return CLIConfig{}, errors.FirstError(errs...)
	}

	if err := cliConfig.validate(); err != nil {
		return CLIConfig{}, errors.Wrap(err, _cliConfigPath)
	}

	return cliConfig, nil
}

func addProfileToCLIConfig(newProfile Profile) error {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return err
	}

	replaced := false
	for i, prevProfile := range cliConfig.Profiles {
		if prevProfile.Name == newProfile.Name {
			cliConfig.Profiles[i] = &newProfile
			replaced = true
			break
		}
	}

	if !replaced {
		cliConfig.Profiles = append(cliConfig.Profiles, &newProfile)
	}

	cliConfig.validate()

	cliConfigBytes, err := yaml.Marshal(cliConfig)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := files.WriteFile(cliConfigBytes, _cliConfigPath); err != nil {
		return err
	}

	return nil
}

func removeProfileFromCLIConfig(profileName string) error {
	cliConfig, err := readCLIConfig()
	if err != nil {
		return err
	}

	var updatedProfiles []*Profile
	deleted := false
	for _, profile := range cliConfig.Profiles {
		if profile.Name == profileName {
			deleted = true
			continue
		}
		updatedProfiles = append(updatedProfiles, profile)
	}

	if deleted == false && profileName != Local.String() {
		return ErrorProfileNotConfigured(profileName)
	}

	cliConfig.Profiles = updatedProfiles

	cliConfig.validate()

	cliConfigBytes, err := yaml.Marshal(cliConfig)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := files.WriteFile(cliConfigBytes, _cliConfigPath); err != nil {
		return err
	}

	return nil
}
