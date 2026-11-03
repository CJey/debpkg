// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package targzip

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTarMode(t *testing.T) {
	for m, want := range map[fs.FileMode]int64{
		0644:                                 0644,
		0755 | fs.ModeSetuid:                 04755,
		0755 | fs.ModeSetgid | fs.ModeSticky: 03755,
		0755 | fs.ModeDir:                    0755,
	} {
		assert.Equal(t, want, tarMode(m), "%v", m)
	}
}
