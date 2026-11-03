// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

//go:build unix

package debpkg

import (
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDataSpecialFile verifies special files are refused, a FIFO without blocking
func TestDataSpecialFile(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "fifo")
	require.NoError(t, syscall.Mkfifo(fifo, 0644))

	d := newData(t)
	assert.ErrorContains(t, d.addFile(fifo, "/fifo"), "not a regular file")
	assert.ErrorContains(t, d.addTree(dir), "unsupported file type")
}
