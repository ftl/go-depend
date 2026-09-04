package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ftl/go-depend/cmd"
)

func TestExitCode(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected int
	}{
		{"no error", nil, 0},
		{"violated invariants", cmd.ErrViolations, 1},
		{"wrapped violated invariants", fmt.Errorf("scan: %w", cmd.ErrViolations), 1},
		{"any other error", errors.New("cannot load packages"), 2},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, exitCode(tc.err))
		})
	}
}
