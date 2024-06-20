package v1alpha1

import (
	rtv1 "github.com/krateoplatformops/provider-runtime/apis/common/v1"
)

// GetCondition of this Definition.
func (mg *Definition) GetCondition(ct rtv1.ConditionType) rtv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetDeletionPolicy of this Definition.
func (mg *Definition) GetDeletionPolicy() rtv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

// SetConditions of this Definition.
func (mg *Definition) SetConditions(c ...rtv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetDeletionPolicy of this Definition.
func (mg *Definition) SetDeletionPolicy(r rtv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}
