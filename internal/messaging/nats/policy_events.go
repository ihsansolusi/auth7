package nats

import "time"

const (
	SubjectPolicyParamsUpdated = "policy7.params.updated"
	SubjectPolicyParamsDeleted = "policy7.params.deleted"
)

type PolicyParamUpdatedEvent struct {
	OrgID         string    `json:"org_id"`
	ParameterName string    `json:"parameter_name"`
	ParameterType string    `json:"parameter_type"`
	UpdatedBy     string    `json:"updated_by"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PolicyParamDeletedEvent struct {
	OrgID         string    `json:"org_id"`
	ParameterName string    `json:"parameter_name"`
	ParameterType string    `json:"parameter_type"`
	DeletedBy     string    `json:"deleted_by"`
	DeletedAt     time.Time `json:"deleted_at"`
}
