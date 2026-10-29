package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeAssignmentNotFound        domainerr.Code = "DOMAIN_AUTHZ_ASSIGNMENT_NOT_FOUND"
	CodeAssignmentDuplicate       domainerr.Code = "DOMAIN_AUTHZ_ASSIGNMENT_DUPLICATE"
	CodeAssignmentNotMember       domainerr.Code = "DOMAIN_AUTHZ_ASSIGNMENT_NOT_MEMBER"
	CodeAssignmentRoleUnavailable domainerr.Code = "DOMAIN_AUTHZ_ASSIGNMENT_ROLE_UNAVAILABLE"
	CodeAssignmentNotSubscribed   domainerr.Code = "DOMAIN_AUTHZ_ASSIGNMENT_NOT_SUBSCRIBED"
	CodeScopeTypeInvalid          domainerr.Code = "DOMAIN_AUTHZ_SCOPE_TYPE_INVALID"
	CodeScopeInvalid              domainerr.Code = "DOMAIN_AUTHZ_SCOPE_INVALID"
	CodeCheckTargetRequired       domainerr.Code = "DOMAIN_AUTHZ_CHECK_TARGET_REQUIRED"
	CodeCheckSolutionUnavailable  domainerr.Code = "DOMAIN_AUTHZ_CHECK_SOLUTION_UNAVAILABLE"
	// CodeSolutionRequired — token user wajib memilih solution_id (token service memakai solution client).
	CodeSolutionRequired       domainerr.Code = "DOMAIN_AUTHZ_SOLUTION_REQUIRED"
	CodeAuthzPersistenceFailed domainerr.Code = "DOMAIN_AUTHZ_PERSISTENCE_FAILED"
	CodeAuthzQueryFailed       domainerr.Code = "DOMAIN_AUTHZ_QUERY_FAILED"
)

// MaxBatchChecks — batas jumlah cek per POST /authz/check-batch.
const MaxBatchChecks = 100
