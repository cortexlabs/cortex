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

package spec

import (
	"fmt"
	"regexp"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

const (
	ErrMalformedConfig              = "spec.malformed_config"
	ErrNoAPIs                       = "spec.no_apis"
	ErrDuplicateName                = "spec.duplicate_name"
	ErrDuplicateEndpointInOneDeploy = "spec.duplicate_endpoint_in_one_deploy"
	ErrDuplicateEndpoint            = "spec.duplicate_endpoint"
	ErrDuplicateContainerName       = "spec.duplicate_container_name"
	ErrConflictingFields            = "spec.conflicting_fields"
	ErrSpecifyOnlyOneField          = "spec.specify_only_one_field"
	ErrSpecifyOneOrTheOther         = "spec.specify_one_or_the_other"
	ErrSpecifyAllOrNone             = "spec.specify_all_or_none"
	ErrOneOfPrerequisitesNotDefined = "spec.one_of_prerequisites_not_defined"
	ErrConfigGreaterThanOtherConfig = "spec.config_greater_than_other_config"

	ErrMinReplicasGreaterThanMax  = "spec.min_replicas_greater_than_max"
	ErrInitReplicasGreaterThanMax = "spec.init_replicas_greater_than_max"
	ErrInitReplicasLessThanMin    = "spec.init_replicas_less_than_min"

	ErrInvalidSurgeOrUnavailable   = "spec.invalid_surge_or_unavailable"
	ErrSurgeAndUnavailableBothZero = "spec.surge_and_unavailable_both_zero"

	ErrShmSizeCannotExceedMem = "spec.shm_size_cannot_exceed_mem"

	ErrFieldMustBeSpecifiedForKind    = "spec.field_must_be_specified_for_kind"
	ErrFieldIsNotSupportedForKind     = "spec.field_is_not_supported_for_kind"
	ErrCortexPrefixedEnvVarNotAllowed = "spec.cortex_prefixed_env_var_not_allowed"
	ErrDisallowedEnvVars              = "spec.disallowed_env_vars"
	ErrRegistryInDifferentRegion      = "spec.registry_in_different_region"
	ErrRegistryAccountIDMismatch      = "spec.registry_account_id_mismatch"
	ErrComputeResourceConflict        = "spec.compute_resource_conflict"
	ErrInvalidNumberOfInfs            = "spec.invalid_number_of_infs"
	ErrIncorrectTrafficSplitterWeight = "spec.incorrect_traffic_splitter_weight"
	ErrTrafficSplitterAPIsNotUnique   = "spec.traffic_splitter_apis_not_unique"
	ErrOneShadowPerTrafficSplitter    = "spec.one_shadow_per_traffic_splitter"
	ErrUnexpectedDockerSecretData     = "spec.unexpected_docker_secret_data"
)

func ErrorMalformedConfig() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMalformedConfig,
		Message: fmt.Sprintf("cortex YAML configuration files must contain a list of maps (see https://docs.cortex.dev/v/%s/ for api configuration schema)", consts.CortexVersionMinor),
	})
}

func ErrorNoAPIs() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrNoAPIs,
		Message: fmt.Sprintf("at least one API must be configured (see https://docs.cortex.dev/v/%s/ for api configuration schema)", consts.CortexVersionMinor),
	})
}

func ErrorDuplicateName(apis []userconfig.API) error {
	filePaths := strset.New()
	for _, api := range apis {
		filePaths.Add(api.FileName)
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateName,
		Message: fmt.Sprintf("name %s must be unique across apis (defined in %s)", s.UserStr(apis[0].Name), s.StrsAnd(filePaths.Slice())),
	})
}

func ErrorDuplicateEndpointInOneDeploy(apis []userconfig.API) error {
	names := make([]string, len(apis))
	for i, api := range apis {
		names[i] = api.Name
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateEndpointInOneDeploy,
		Message: fmt.Sprintf("endpoint %s must be unique across apis (defined in %s)", s.UserStr(*apis[0].Networking.Endpoint), s.StrsAnd(names)),
	})
}

func ErrorDuplicateEndpoint(apiName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateEndpoint,
		Message: fmt.Sprintf("endpoint is already being used by %s", apiName),
	})
}

func ErrorDuplicateContainerName(containerName string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDuplicateContainerName,
		Message: fmt.Sprintf("container name %s must be unique", containerName),
	})
}

func ErrorConflictingFields(fieldKeyA, fieldKeyB string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrConflictingFields,
		Message: fmt.Sprintf("please specify either the %s or %s field (both cannot be specified at the same time)", fieldKeyA, fieldKeyB),
	})
}

func ErrorSpecifyOnlyOneField(fields ...string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyOnlyOneField,
		Message: fmt.Sprintf("please specify only one of the following fields %s", s.UserStrsOr(fields)),
	})
}

func ErrorSpecifyOneOrTheOther(fieldKeyA, fieldKeyB string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyOneOrTheOther,
		Message: fmt.Sprintf("please specify either the %s field or %s field (cannot be both empty at the same time)", fieldKeyA, fieldKeyB),
	})
}

func ErrorSpecifyAllOrNone(val string, vals ...string) error {
	allVals := append([]string{val}, vals...)
	message := fmt.Sprintf("please specify all or none of %s", s.UserStrsAnd(allVals))
	if len(allVals) == 2 {
		message = fmt.Sprintf("please specify both %s and %s or neither of them", s.UserStr(allVals[0]), s.UserStr(allVals[1]))
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrSpecifyAllOrNone,
		Message: message,
	})
}

func ErrorOneOfPrerequisitesNotDefined(argName string, prerequisite string, prerequisites ...string) error {
	allPrerequisites := append([]string{prerequisite}, prerequisites...)
	message := fmt.Sprintf("%s specified without specifying %s", s.UserStr(argName), s.UserStrsOr(allPrerequisites))

	return errors.WithStack(&errors.Error{
		Kind:    ErrOneOfPrerequisitesNotDefined,
		Message: message,
	})
}

func ErrorConfigGreaterThanOtherConfig(tooBigKey string, tooBigVal interface{}, tooSmallKey string, tooSmallVal interface{}) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrConfigGreaterThanOtherConfig,
		Message: fmt.Sprintf("%s (%s) cannot be greater than %s (%s)", tooBigKey, s.UserStr(tooBigVal), tooSmallKey, s.UserStr(tooSmallVal)),
	})
}

func ErrorMinReplicasGreaterThanMax(min int32, max int32) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrMinReplicasGreaterThanMax,
		Message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", userconfig.MinReplicasKey, userconfig.MaxReplicasKey, min, max),
	})
}

func ErrorInitReplicasGreaterThanMax(init int32, max int32) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInitReplicasGreaterThanMax,
		Message: fmt.Sprintf("%s cannot be greater than %s (%d > %d)", userconfig.InitReplicasKey, userconfig.MaxReplicasKey, init, max),
	})
}

func ErrorInitReplicasLessThanMin(init int32, min int32) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInitReplicasLessThanMin,
		Message: fmt.Sprintf("%s cannot be less than %s (%d < %d)", userconfig.InitReplicasKey, userconfig.MinReplicasKey, init, min),
	})
}

func ErrorInvalidSurgeOrUnavailable(val string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidSurgeOrUnavailable,
		Message: fmt.Sprintf("%s is not a valid value - must be an integer percentage (e.g. 25%%, to denote a percentage of desired replicas) or a positive integer (e.g. 5, to denote a number of replicas)", s.UserStr(val)),
	})
}

func ErrorSurgeAndUnavailableBothZero() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrSurgeAndUnavailableBothZero,
		Message: fmt.Sprintf("%s and %s cannot both be zero", userconfig.MaxSurgeKey, userconfig.MaxUnavailableKey),
	})
}

func ErrorShmSizeCannotExceedMem(shmSize k8s.Quantity, mem k8s.Quantity) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrShmSizeCannotExceedMem,
		Message: fmt.Sprintf("shm_size (%s) cannot exceed total compute mem (%s)", shmSize.UserString, mem.UserString),
	})
}

func ErrorFieldMustBeSpecifiedForKind(field string, kind userconfig.Kind) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFieldMustBeSpecifiedForKind,
		Message: fmt.Sprintf("field %s must be specified for %s kind", field, kind.String()),
	})
}

func ErrorFieldIsNotSupportedForKind(field string, kind userconfig.Kind) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrFieldIsNotSupportedForKind,
		Message: fmt.Sprintf("%s field is not supported for %s kind", field, kind.String()),
	})
}

func ErrorCortexPrefixedEnvVarNotAllowed(prefixes ...string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrCortexPrefixedEnvVarNotAllowed,
		Message: fmt.Sprintf("environment variables starting with %s are reserved", s.StrsOr(prefixes)),
	})
}

func ErrorDisallowedEnvVars(disallowedValues ...string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrDisallowedEnvVars,
		Message: fmt.Sprintf("environment %s %s %s disallowed", s.PluralS("variables", len(disallowedValues)), s.StrsAnd(disallowedValues), s.PluralToBe(len(disallowedValues))),
	})
}

func ErrorRegistryInDifferentRegion(registryRegion string, awsClientRegion string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrRegistryInDifferentRegion,
		Message: fmt.Sprintf("registry region (%s) does not match cortex's region (%s); images can only be pulled from repositories in the same region as cortex", registryRegion, awsClientRegion),
	})
}

func ErrorRegistryAccountIDMismatch(regID, opID string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrRegistryAccountIDMismatch,
		Message: fmt.Sprintf("registry account ID (%s) doesn't match your AWS account ID (%s), and using an ECR registry in a different AWS account is not supported", regID, opID),
	})
}

func ErrorComputeResourceConflict(resourceA, resourceB string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrComputeResourceConflict,
		Message: fmt.Sprintf("%s and %s resources cannot be used together", resourceA, resourceB),
	})
}

func ErrorInvalidNumberOfInfs(requestedInfs int64) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrInvalidNumberOfInfs,
		Message: fmt.Sprintf("cannot request %d Infs (currently only 1 Inf can be used per API replica, due to AWS's bug: https://github.com/aws/aws-neuron-sdk/issues/110)", requestedInfs),
	})
}

func ErrorIncorrectTrafficSplitterWeightTotal(totalWeight int32) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrIncorrectTrafficSplitterWeight,
		Message: fmt.Sprintf("expected weights of all non-shadow apis to sum to 100 but found %d", totalWeight),
	})
}

func ErrorTrafficSplitterAPIsNotUnique(names []string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrTrafficSplitterAPIsNotUnique,
		Message: fmt.Sprintf("%s not unique: %s", s.PluralS("api", len(names)), s.StrsSentence(names, "")),
	})
}

func ErrorOneShadowPerTrafficSplitter() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrOneShadowPerTrafficSplitter,
		Message: "multiple shadow apis detected; only one api is allowed to be marked as a shadow",
	})
}

var _pwRegex = regexp.MustCompile(`"password":"[^"]+"`)
var _authRegex = regexp.MustCompile(`"auth":"[^"]+"`)

func ErrorUnexpectedDockerSecretData(reason string, secretData map[string][]byte) error {
	secretDataStrMap := map[string]string{}

	for key, value := range secretData {
		valueStr := string(value)
		valueStr = _pwRegex.ReplaceAllString(valueStr, `"password":"<omitted>"`)
		valueStr = _authRegex.ReplaceAllString(valueStr, `"auth":"<omitted>"`)
		secretDataStrMap[key] = valueStr
	}

	return errors.WithStack(&errors.Error{
		Kind:    ErrUnexpectedDockerSecretData,
		Message: fmt.Sprintf("docker registry secret named \"%s\" was found, but contains unexpected data (%s); got: %s", _dockerPullSecretName, reason, s.UserStr(secretDataStrMap)),
	})
}
