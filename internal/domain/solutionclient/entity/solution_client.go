package entity

import (
	"crypto/subtle"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	domainerr "platform-service/internal/domain/errors"
	"platform-service/internal/domain/solutionclient/constant"
	"platform-service/internal/domain/solutionclient/valueobject"
)

// SolutionClient adalah kredensial client credentials milik satu solution.
type SolutionClient struct {
	ID               string
	SolutionID       string
	Name             string
	ClientID         string
	ClientSecretHash string
	Scopes           valueobject.Scopes
	Status           valueobject.ClientStatus
	LastUsedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	RevokedAt        *time.Time
}

type NewSolutionClientParams struct {
	SolutionID       string
	Name             string
	ClientID         string // dibuat usecase (acak)
	ClientSecretHash string
	Scopes           []string
}

func NewSolutionClient(p NewSolutionClientParams, now time.Time) (*SolutionClient, error) {
	name := strings.TrimSpace(p.Name)
	if name == "" || utf8.RuneCountInString(name) > constant.NameMaxLength {
		return nil, domainerr.New(constant.CodeClientNameInvalid)
	}
	scopes, err := valueobject.NewScopes(p.Scopes)
	if err != nil {
		return nil, err
	}
	if p.SolutionID == "" || p.ClientID == "" || p.ClientSecretHash == "" {
		return nil, domainerr.New(constant.CodeClientInvalid)
	}
	now = now.UTC()
	return &SolutionClient{
		ID:               uuid.NewString(),
		SolutionID:       p.SolutionID,
		Name:             name,
		ClientID:         p.ClientID,
		ClientSecretHash: p.ClientSecretHash,
		Scopes:           scopes,
		Status:           valueobject.ClientStatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func (c *SolutionClient) IsActive() bool {
	return c.Status == valueobject.ClientStatusActive
}

// SecretMatches membandingkan hash secret secara constant-time.
func (c *SolutionClient) SecretMatches(secretHash string) bool {
	return subtle.ConstantTimeCompare([]byte(c.ClientSecretHash), []byte(secretHash)) == 1
}

// RotateSecret mengganti secret; secret lama langsung tidak berlaku.
func (c *SolutionClient) RotateSecret(newHash string, now time.Time) error {
	if !c.IsActive() {
		return domainerr.New(constant.CodeClientRevoked)
	}
	if newHash == "" {
		return domainerr.New(constant.CodeClientInvalid)
	}
	c.ClientSecretHash = newHash
	c.UpdatedAt = now.UTC()
	return nil
}

// Revoke menonaktifkan client secara permanen. Idempotent.
func (c *SolutionClient) Revoke(now time.Time) {
	if !c.IsActive() {
		return
	}
	t := now.UTC()
	c.Status = valueobject.ClientStatusRevoked
	c.RevokedAt = &t
	c.UpdatedAt = t
}

// GrantableScopes menentukan scope token: kosong → semua scope client;
// bila diminta, harus subset scope client.
func (c *SolutionClient) GrantableScopes(requested valueobject.Scopes) (valueobject.Scopes, error) {
	if requested.IsEmpty() {
		return c.Scopes, nil
	}
	if !requested.IsSubsetOf(c.Scopes) {
		return valueobject.Scopes{}, domainerr.New(constant.CodeClientScopeInvalid)
	}
	return requested, nil
}
