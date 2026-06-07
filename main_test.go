package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitLoggerInitializesGlobalLogger(t *testing.T) {
	old := infoLogger
	infoLogger = nil
	t.Cleanup(func() {
		infoLogger = old
	})

	err := initLogger()
	require.NoError(t, err)
	require.NotNil(t, infoLogger)
}
