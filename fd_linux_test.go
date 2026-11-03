// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"os"
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjey/debpkg/internal/test"
)

// TestNoFileDescriptorLeak runs without GC, whose finalizers would close leaked files
func TestNoFileDescriptorLeak(t *testing.T) {
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	count := func() int {
		entries, err := os.ReadDir("/proc/self/fd")
		require.NoError(t, err)
		return len(entries)
	}

	before := count()
	for range 10 {
		deb := New()
		setMandatory(deb, "fd")
		require.NoError(t, deb.AddFileString("x", "/x"))
		require.NoError(t, deb.Write(test.TempFile(t)))
		require.NoError(t, New().Close())
	}
	assert.Equal(t, before, count())
}
