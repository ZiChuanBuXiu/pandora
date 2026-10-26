// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package dataworkarounds

import (
	"reflect"
	"testing"

	sdkModels "github.com/hashicorp/pandora/tools/data-api-sdk/v1/models"
)

func TestWorkaroundHorizonDBParameterGroupOptionalIsApplicable(t *testing.T) {
	tests := []struct {
		name       string
		service    string
		apiVersion string
		expected   bool
	}{
		{"matching service and version", "HorizonDB", "2026-10-01-preview", true},
		{"different service", "MongoCluster", "2026-10-01-preview", false},
		{"different version", "HorizonDB", "2026-10-01", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := (workaroundHorizonDBParameterGroupOptional{}).IsApplicable(test.service, sdkModels.APIVersion{APIVersion: test.apiVersion})
			if actual != test.expected {
				t.Fatalf("expected applicability %t, got %t", test.expected, actual)
			}
		})
	}
}

func TestWorkaroundHorizonDBParameterGroupOptionalApply(t *testing.T) {
	referenceName := "PoolParameterGroupReference"
	field := sdkModels.SDKField{
		JsonName: "parameterGroup",
		Required: true,
		ObjectDefinition: sdkModels.SDKObjectDefinition{
			Type:          sdkModels.ReferenceSDKObjectDefinitionType,
			ReferenceName: &referenceName,
		},
	}
	otherField := sdkModels.SDKField{JsonName: "nodeCount", Required: true}
	patchModel := sdkModels.SDKModel{
		Fields: map[string]sdkModels.SDKField{
			"ParameterGroup": {JsonName: "parameterGroup", Optional: true},
		},
	}
	input := sdkModels.APIVersion{
		APIVersion: "2026-10-01-preview",
		Resources: map[string]sdkModels.APIResource{
			"Pools": {
				Models: map[string]sdkModels.SDKModel{
					"PoolProperties": {
						Fields: map[string]sdkModels.SDKField{
							"ParameterGroup": field,
							"NodeCount":      otherField,
						},
					},
					"PoolPropertiesForPatchUpdate": patchModel,
				},
			},
		},
	}

	output, err := Apply("HorizonDB", input)
	if err != nil {
		t.Fatal(err)
	}

	field.Optional = true
	field.Required = false
	actual := output.Resources["Pools"].Models["PoolProperties"].Fields
	if !reflect.DeepEqual(actual["ParameterGroup"], field) {
		t.Fatalf("expected only optional/required flags to change, got %#v", actual["ParameterGroup"])
	}
	if !reflect.DeepEqual(actual["NodeCount"], otherField) {
		t.Fatal("unrelated field changed")
	}
	if !reflect.DeepEqual(output.Resources["Pools"].Models["PoolPropertiesForPatchUpdate"], patchModel) {
		t.Fatal("PATCH model changed")
	}

	if _, err := Apply("HorizonDB", *output); err != nil {
		t.Fatalf("reapplying workaround failed: %s", err)
	}
}

func TestWorkaroundHorizonDBParameterGroupOptionalMissingSchema(t *testing.T) {
	tests := []struct {
		name      string
		resources map[string]sdkModels.APIResource
		expected  string
	}{
		{
			name:     "missing resource",
			expected: "expected a resource named `Pools` but didn't get one",
		},
		{
			name:      "missing model",
			resources: map[string]sdkModels.APIResource{"Pools": {}},
			expected:  "couldn't find model `PoolProperties`",
		},
		{
			name: "missing field",
			resources: map[string]sdkModels.APIResource{
				"Pools": {Models: map[string]sdkModels.SDKModel{"PoolProperties": {}}},
			},
			expected: "couldn't find the field `ParameterGroup` within model `PoolProperties`",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := (workaroundHorizonDBParameterGroupOptional{}).Process(sdkModels.APIVersion{Resources: test.resources})
			if err == nil || err.Error() != test.expected {
				t.Fatalf("expected error %q, got %v", test.expected, err)
			}
			if output != nil {
				t.Fatal("expected no output on error")
			}
		})
	}
}
