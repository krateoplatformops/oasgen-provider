package restdefinition

import (
	"testing"

	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestExpandWildcardActions(t *testing.T) {
	allVerbs := []definitionv1alpha1.VerbsDescription{
		{Action: "create"},
		{Action: "get"},
		{Action: "update"},
		{Action: "delete"},
	}

	testCases := []struct {
		name           string
		actions        []string
		verbs          []definitionv1alpha1.VerbsDescription
		expectedResult []string
		expectedError  bool
	}{
		{
			name:           "Wildcard should expand to all verb actions",
			actions:        []string{"*"},
			verbs:          allVerbs,
			expectedResult: []string{"create", "get", "update", "delete"},
			expectedError:  false,
		},
		{
			name:           "Explicit actions should remain unchanged",
			actions:        []string{"create", "delete"},
			verbs:          allVerbs,
			expectedResult: []string{"create", "delete"},
			expectedError:  false,
		},
		{
			name:           "Empty actions list should remain empty",
			actions:        []string{},
			verbs:          allVerbs,
			expectedResult: []string{},
			expectedError:  false,
		},
		{
			name:           "Nil actions list should remain nil",
			actions:        nil,
			verbs:          allVerbs,
			expectedResult: nil,
			expectedError:  false,
		},
		{
			name:           "Wildcard with no verbs should result in an empty list",
			actions:        []string{"*"},
			verbs:          []definitionv1alpha1.VerbsDescription{},
			expectedResult: []string{},
			expectedError:  false,
		},
		{
			name:           "Actions list with other values alongside wildcard should get error",
			actions:        []string{"*", "get", "update"},
			verbs:          allVerbs,
			expectedResult: nil,
			expectedError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := expandWildcardActions(tc.actions, tc.verbs)

			if tc.expectedError {
				assert.Error(t, err, "Expected an error but got none")
				assert.Nil(t, result, "Result should be nil when error occurs")
			} else {
				assert.NoError(t, err, "Unexpected error occurred")
				assert.Equal(t, tc.expectedResult, result, "The expanded actions did not match the expected result")
			}
		})
	}
}
