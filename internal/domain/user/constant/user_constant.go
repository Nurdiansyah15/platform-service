package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeUserNotFound          domainerr.Code = "DOMAIN_USER_NOT_FOUND"
	CodeUserEmailDuplicate    domainerr.Code = "DOMAIN_USER_EMAIL_DUPLICATE"
	CodeUserEmailInvalid      domainerr.Code = "DOMAIN_USER_EMAIL_INVALID"
	CodeUserNameInvalid       domainerr.Code = "DOMAIN_USER_NAME_INVALID"
	CodeUserPasswordInvalid   domainerr.Code = "DOMAIN_USER_PASSWORD_INVALID"
	CodeUserStatusInvalid     domainerr.Code = "DOMAIN_USER_STATUS_INVALID"
	CodeUserPersistenceFailed domainerr.Code = "DOMAIN_USER_PERSISTENCE_FAILED"
	CodeUserQueryFailed       domainerr.Code = "DOMAIN_USER_QUERY_FAILED"
)

const (
	EmailMaxLength    = 255
	NameMaxLength     = 255
	PasswordMinLength = 8
	// PasswordMaxLength — bcrypt hanya memakai 72 byte pertama.
	PasswordMaxLength = 72
)
