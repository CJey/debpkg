// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteFailureKeepsSpecialOutput verifies a failed Write removes neither a device, as
// /dev/stdout would be when running as root, nor a link to it
func TestWriteFailureKeepsSpecialOutput(t *testing.T) {
	// Otherwise, as root, Write would create a regular /dev/full
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("no /dev/full")
	}
	out := filepath.Join(t.TempDir(), "full.deb")
	require.NoError(t, os.Symlink("/dev/full", out))
	deb := newTestDeb(t, "full")
	assert.ErrorContains(t, deb.Write(out), "no space left on device")
	_, err := os.Lstat(out)
	assert.NoError(t, err)
}
