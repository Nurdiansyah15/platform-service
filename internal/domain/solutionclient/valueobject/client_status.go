package valueobject

import (
	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solutionclient/constant"
)

type ClientStatus string

const (
	ClientStatusActive  ClientStatus = "active"
	ClientStatusRevoked ClientStatus = "revoked"
)

func NewClientStatus(raw string) (ClientStatus, error) {
	switch s := ClientStatus(raw); s {
	case ClientStatusActive, ClientStatusRevoked:
		return s, nil
	default:
		return "", domainerr.New(constant.CodeClientInvalid)
	}
}

func (s ClientStatus) String() string { return string(s) }
