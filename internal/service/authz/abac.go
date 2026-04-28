package authz

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ihsansolusi/auth7/internal/domain"
)

type ABACEvaluator struct {
	policyStore ABACPolicyStore
}

func NewABACEvaluator(policyStore ABACPolicyStore) *ABACEvaluator {
	return &ABACEvaluator{
		policyStore: policyStore,
	}
}

func (e *ABACEvaluator) Evaluate(ctx context.Context, authCtx *domain.AuthContext, permission string, resource interface{}) (*domain.AuthorizationResult, error) {
	policies, err := e.policyStore.ListByOrg(ctx, authCtx.OrgID)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}

	var applicablePolicies []*domain.ABACPolicy
	for _, p := range policies {
		if e.matchesPermission(p, permission) && e.matchesConditions(p.Conditions, authCtx) {
			applicablePolicies = append(applicablePolicies, p)
		}
	}

	var fieldMasks []domain.FieldMask
	denied := false

	for _, p := range applicablePolicies {
		if p.Effect == domain.ABACEffectDeny {
			denied = true
		}
		if len(p.Fields) > 0 {
			for _, field := range p.Fields {
				fieldMasks = append(fieldMasks, domain.FieldMask{
					Field:     field,
					MaskValue: "***",
					Reason:    p.Description,
				})
			}
		}
	}

	if denied {
		return &domain.AuthorizationResult{
			Allowed:    false,
			Reason:     "denied by ABAC policy",
			FieldMasks: fieldMasks,
		}, nil
	}

	return &domain.AuthorizationResult{
		Allowed:    true,
		Reason:     "allowed by ABAC policy",
		FieldMasks: fieldMasks,
	}, nil
}

func (e *ABACEvaluator) matchesPermission(policy *domain.ABACPolicy, permission string) bool {
	conditions, ok := policy.Conditions["permission"]
	if !ok {
		return true
	}

	switch cond := conditions.(type) {
	case string:
		return cond == permission || cond == "*"
	case []interface{}:
		for _, c := range cond {
			if s, ok := c.(string); ok && (s == permission || s == "*") {
				return true
			}
		}
	}
	return false
}

func (e *ABACEvaluator) matchesConditions(conditions map[string]interface{}, authCtx *domain.AuthContext) bool {
	for key, expected := range conditions {
		if key == "permission" || key == "effect" {
			continue
		}

		var actual interface{}
		switch key {
		case "branch_scope":
			actual = string(authCtx.BranchScope)
		case "user_id":
			actual = authCtx.UserID.String()
		case "org_id":
			actual = authCtx.OrgID.String()
		case "branch_id":
			actual = authCtx.BranchID.String()
		default:
			if authCtx.Attributes != nil {
				actual = authCtx.Attributes[key]
			}
		}

		if !e.compareValues(expected, actual) {
			return false
		}
	}
	return true
}

func (e *ABACEvaluator) compareValues(expected, actual interface{}) bool {
	if expected == "*" {
		return true
	}

	switch exp := expected.(type) {
	case string:
		act, ok := actual.(string)
		return ok && exp == act
	case int:
		act, ok := actual.(int)
		return ok && exp == act
	case float64:
		act, ok := actual.(float64)
		return ok && exp == act
	case bool:
		act, ok := actual.(bool)
		return ok && exp == act
	case []interface{}:
		for _, v := range exp {
			if e.compareValues(v, actual) {
				return true
			}
		}
		return false
	}

	return reflect.DeepEqual(expected, actual)
}

func (e *ABACEvaluator) ApplyFieldMasks(data map[string]interface{}, masks []domain.FieldMask) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		result[k] = v
	}
	for _, m := range masks {
		if _, ok := result[m.Field]; ok {
			result[m.Field] = m.MaskValue
		}
	}
	return result
}