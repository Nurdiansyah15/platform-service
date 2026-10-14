package constant

import domainerr "platform-service/internal/domain/errors"

const (
	CodeSigningKeyNotFound          domainerr.Code = "DOMAIN_SIGNING_KEY_NOT_FOUND"
	CodeSigningKeyActiveExists      domainerr.Code = "DOMAIN_SIGNING_KEY_ACTIVE_EXISTS"
	CodeSigningKeyInvalid           domainerr.Code = "DOMAIN_SIGNING_KEY_INVALID"
	CodeSigningKeyStatusInvalid     domainerr.Code = "DOMAIN_SIGNING_KEY_STATUS_INVALID"
	CodeSigningKeyTransitionInvalid domainerr.Code = "DOMAIN_SIGNING_KEY_TRANSITION_INVALID"
	CodeSigningKeyPersistenceFailed domainerr.Code = "DOMAIN_SIGNING_KEY_PERSISTENCE_FAILED"
	CodeSigningKeyQueryFailed       domainerr.Code = "DOMAIN_SIGNING_KEY_QUERY_FAILED"
)

// AlgorithmRS256 adalah satu-satunya algoritma yang didukung saat ini.
const AlgorithmRS256 = "RS256"
