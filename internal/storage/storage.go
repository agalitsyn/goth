package storage

import (
	"errors"
)

// General errors is returned by storage implementations for handling errors in business logic code.
// Specific storage errors should be mapped to this errors.
// For example:
//   - pgx.ErrNoRows in postgres should be mapped to ErrNotFound
var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate")
)
