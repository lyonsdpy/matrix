package desktop_test

import (
	"errors"
	"testing"

	"matrix/api/internal/infra/desktop"
)

func TestNewDB_EmptyDSN(t *testing.T) {
	_, err := desktop.NewDB("")
	if !errors.Is(err, desktop.ErrMissingDSN) {
		t.Fatalf("expected ErrMissingDSN, got %v", err)
	}
}
