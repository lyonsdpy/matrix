package postgres_test

import (
	"errors"
	"testing"

	"matrix/api/internal/infra/postgres"
)

func TestOpen_EmptyDSN_ReturnsError(t *testing.T) {
	_, err := postgres.Open("")
	if !errors.Is(err, postgres.ErrEmptyDSN) {
		t.Errorf("error = %v, want ErrEmptyDSN", err)
	}
}
