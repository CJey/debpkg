// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"maps"
	"path"
	"strings"
	"text/template"
)

// prefixedVars are the directories expanded below INSTALLPREFIX unless absolute
var prefixedVars = []string{"BINDIR", "SBINDIR", "SYSCONFDIR", "DATAROOTDIR"}

func defaultVars() map[string]string {
	return map[string]string{
		"INSTALLPREFIX": DefaultInstallPrefix,
		"BINDIR":        DefaultBinDir,
		"SBINDIR":       DefaultSbinDir,
		"SYSCONFDIR":    DefaultSysConfDir,
		"DATAROOTDIR":   DefaultDataRootDir,
	}
}

// SetVar sets a variable for use with config file
func (deb *DebPkg) SetVar(key, val string) {
	deb.vars[key] = val
}

// GetVar gets a variable
func (deb *DebPkg) GetVar(key string) string {
	return deb.vars[key]
}

// GetVarWithPrefix gets a variable, prefixed with INSTALLPREFIX when the value is relative
func (deb *DebPkg) GetVarWithPrefix(key string) string {
	val := deb.vars[key]
	if val == "" || strings.HasPrefix(val, "/") {
		return val
	}
	return path.Join("/", deb.vars["INSTALLPREFIX"], val)
}

// ExpandVar expands the variables referenced as {{.NAME}} in s, the directories of the
// default variables with GetVarWithPrefix. An undefined variable is an error.
func (deb *DebPkg) ExpandVar(s string) (string, error) {
	tmpl, err := template.New("").Option("missingkey=error").Parse(s)
	if err != nil {
		return "", err
	}
	vars := maps.Clone(deb.vars)
	for _, key := range prefixedVars {
		vars[key] = deb.GetVarWithPrefix(key)
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, vars); err != nil {
		return "", err
	}
	return b.String(), nil
}
