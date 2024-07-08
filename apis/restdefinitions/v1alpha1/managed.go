package v1alpha1

import (
	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
)

// GetCondition of this RestDefinition.
func (mg *RestDefinition) GetCondition(ct rtv1.ConditionType) rtv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetDeletionPolicy of this RestDefinition.
func (mg *RestDefinition) GetDeletionPolicy() rtv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

// SetConditions of this RestDefinition.
func (mg *RestDefinition) SetConditions(c ...rtv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetDeletionPolicy of this RestDefinition.
func (mg *RestDefinition) SetDeletionPolicy(r rtv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}
