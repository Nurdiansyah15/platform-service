package valueobject

import (
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/user/constant"
)

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
)

func NewUserStatus(raw string) (UserStatus, error) {
	switch s := UserStatus(raw); s {
	case UserStatusActive, UserStatusInactive, UserStatusSuspended:
		return s, nil
	default:
		return "", domainerr.New(constant.CodeUserStatusInvalid)
	}
}

func (s UserStatus) String() string { return string(s) }
