// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"os"
	"runtime"
	"strings"
)

// debianArch maps the GOARCH values whose Debian names differ, arm aside
var debianArch = map[string]string{
	"386":      "i386",
	"mipsle":   "mipsel",
	"mips64le": "mips64el",
	"ppc64le":  "ppc64el",
}

// GetArchitecture returns the Debian architecture of $GOARCH, or of runtime.GOARCH when unset.
// GOARCH=arm maps to armel for GOARM=5, otherwise to armhf, which Raspberry Pi OS also uses
// for ARMv6.
func GetArchitecture() string {
	goarch := os.Getenv("GOARCH")
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if goarch == "arm" {
		if strings.HasPrefix(os.Getenv("GOARM"), "5") {
			return "armel"
		}
		return "armhf"
	}
	if arch, ok := debianArch[goarch]; ok {
		return arch
	}
	return goarch
}
