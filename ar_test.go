// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArWriter(t *testing.T) {
	var buf bytes.Buffer
	ar := arWriter{w: &buf, mtime: 1700000000}
	require.NoError(t, ar.add("odd", 3, strings.NewReader("abc")))
	require.NoError(t, ar.add("even", 2, strings.NewReader("de")))

	// name 16, mtime 12, uid 6, gid 6, mode 8, size 10, "`\n"; odd sizes padded with "\n"
	expect := "odd" + strings.Repeat(" ", 13) + "1700000000  " + "0     " + "0     " + "100644  " +
		"3" + strings.Repeat(" ", 9) + "`\n" + "abc\n" +
		"even" + strings.Repeat(" ", 12) + "1700000000  " + "0     " + "0     " + "100644  " +
		"2" + strings.Repeat(" ", 9) + "`\n" + "de"
	assert.Equal(t, expect, buf.String())

	buf.Reset()
	assert.Error(t, ar.add("name-longer-than-16", 0, strings.NewReader("")))
	assert.Empty(t, buf.String(), "nothing written for a header that does not fit")
	assert.Error(t, ar.add("short", 5, strings.NewReader("abc")))
}
