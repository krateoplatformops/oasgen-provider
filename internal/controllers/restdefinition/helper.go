package restdefinition

import (
	"fmt"

	definitionv1alpha1 "github.com/krateoplatformops/oasgen-provider/apis/restdefinitions/v1alpha1"
)

// expandWildcardActions expands "*" wildcard to all available verb actions
func expandWildcardActions(actions []string, verbsDescription []definitionv1alpha1.VerbsDescription) ([]string, error) {
	// Check for mixed wildcard usage first
	hasWildcard := false
	hasOthers := false
	for _, action := range actions {
		if action == "*" {
			hasWildcard = true
		} else {
			hasOthers = true
		}
	}

	if hasWildcard && hasOthers {
		return nil, fmt.Errorf("invalid configuration: '*' wildcard cannot be mixed with specific actions in the list")
	}

	if hasWildcard {
		expandedActions := make([]string, 0, len(verbsDescription))
		for _, verb := range verbsDescription {
			expandedActions = append(expandedActions, verb.Action)
		}
		return expandedActions, nil
	}

	return actions, nil
}