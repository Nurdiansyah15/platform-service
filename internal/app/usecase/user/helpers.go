package user

import (
	"errors"

	"platform-service/internal/app/apperror"
	"platform-service/internal/app/dto"
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/user/constant"
	"platform-service/internal/domain/user/entity"
)

func mapUserDomainError(err error) error {
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		return apperror.Internal(string(apperror.CodeInternal), err)
	}
	switch de.Code {
	case constant.CodeUserNotFound:
		return apperror.NotFound(string(de.Code))
	case constant.CodeUserSelfChangeForbidden:
		return apperror.Forbidden(string(de.Code))
	case constant.CodeUserEmailDuplicate:
		return apperror.Conflict(string(de.Code), err)
	case constant.CodeUserEmailInvalid, constant.CodeUserNameInvalid,
		constant.CodeUserPasswordInvalid, constant.CodeUserStatusInvalid, constant.CodeUserCurrentPasswordInvalid:
		return apperror.Unprocessable(string(de.Code), nil, err)
	default:
		return apperror.Internal(string(de.Code), err)
	}
}

func isDomainCode(err error, code domainerr.Code) bool {
	var de *domainerr.DomainError
	return errors.As(err, &de) && de.Code == code
}

// ToUserResponse dipakai juga oleh usecase auth (GET /auth/me).
func ToUserResponse(u *entity.User) dto.UserResponse {
	return dto.UserResponse{
		ID:              u.ID,
		Email:           u.Email.Value(),
		Name:            u.Name,
		Status:          u.Status.String(),
		IsPlatformAdmin: u.IsPlatformAdmin,
		HasPassword:     u.HasPassword(),
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}
}
