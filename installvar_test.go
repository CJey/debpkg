// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVarInit(t *testing.T) {
	deb := New()
	defer deb.Close()

	tvs := map[string]string{
		"INSTALLPREFIX": DefaultInstallPrefix,
		"BINDIR":        DefaultBinDir,
		"SBINDIR":       DefaultSbinDir,
		"SYSCONFDIR":    DefaultSysConfDir,
		"DATAROOTDIR":   DefaultDataRootDir,
	}
	for v, exp := range tvs {
		assert.Equal(t, exp, deb.GetVar(v))
	}
	assert.Empty(t, deb.GetVar("UNSET"))
}

func TestGetVarWithPrefix(t *testing.T) {
	deb := New()
	defer deb.Close()

	tvs := map[string]string{
		"BINDIR":      "/usr/bin",
		"SBINDIR":     "/usr/sbin",
		"SYSCONFDIR":  "/etc",
		"DATAROOTDIR": "/usr/share",
		"UNSET":       "",
	}
	for v, exp := range tvs {
		assert.Equal(t, exp, deb.GetVarWithPrefix(v), v)
	}

	deb.SetVar("INSTALLPREFIX", "/opt/foo")
	assert.Equal(t, "/opt/foo/bin", deb.GetVarWithPrefix("BINDIR"))
	deb.SetVar("INSTALLPREFIX", "/")
	assert.Equal(t, "/bin", deb.GetVarWithPrefix("BINDIR"))
}

func TestExpandVar(t *testing.T) {
	deb := New()
	defer deb.Close()

	tvs := map[string]string{
		"{{.INSTALLPREFIX}}": "/usr",
		"{{.BINDIR}}":        "/usr/bin",
		"{{.SBINDIR}}":       "/usr/sbin",
		"{{.SYSCONFDIR}}":    "/etc",
		"{{.DATAROOTDIR}}":   "/usr/share",
	}
	for val, exp := range tvs {
		res, err := deb.ExpandVar(val)
		assert.NoError(t, err)
		assert.Equal(t, exp, res)
	}

	deb.SetVar("APPNAME", "foo")
	res, err := deb.ExpandVar("{{.DATAROOTDIR}}/{{.APPNAME}}")
	assert.NoError(t, err)
	assert.Equal(t, "/usr/share/foo", res)
	_, err = deb.ExpandVar("{{.UNDEFINED}}")
	assert.Error(t, err)

	// Variables belong to their package
	other := New()
	defer other.Close()
	_, err = other.ExpandVar("{{.APPNAME}}")
	assert.Error(t, err)
}
