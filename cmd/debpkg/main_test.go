package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMainHappyFlow(t *testing.T) {
	configFile = "/dev/null"
	f, err := os.CreateTemp("", "debpkg")
	require.Nil(t, err)
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()
	outputFile = f.Name()
	main()
	// we should get here without fatal errors
}
