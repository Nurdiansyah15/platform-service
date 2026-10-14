package testhelper

import (
	"errors"
	"testing"

	domainerr "platform-service/internal/domain/errors"
)

// AssertDomainError checks that err is a *DomainError with the given code.
func AssertDomainError(t *testing.T, err error, wantCode domainerr.Code) {
	t.Helper()
	var de *domainerr.DomainError
	if !errors.As(err, &de) {
		t.Fatalf("expected *DomainError with code %q, got: %v (type %T)", wantCode, err, err)
	}
	if de.Code != wantCode {
		t.Fatalf("expected domain error code %q, got %q (full err: %v)", wantCode, de.Code, err)
	}
}
