// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"fmt"
	"testing"
)

// Test correct output of a empty control file when no DepPkg Set* functions are called
// Only the mandatory fields are exported then, this behaviour is checked
func TestControlFileEmpty(t *testing.T) {
	controlExpect := `Package: 
Version: 0.0.0
Architecture: amd64
Maintainer:  <>
Installed-Size: 0
Description: 
`
	// Empty
	deb := New()
	defer deb.Close()

	// architecture is auto-set when empty, this makes sure it is always set to amd64
	deb.SetArchitecture("amd64")
	control := deb.control.String(0)

	if control != controlExpect {
		t.Error("Unexpected control file")
		fmt.Printf("--- expected (len %d):\n'%s'\n--- got (len %d):\n'%s'---\n", len(controlExpect), controlExpect, len(control), control)
	}
}

// Test correct output of a control file when SetVcs* functions are called
// Only the mandatory fields are exported then, this behaviour is checked
func TestControlFileVcsAndVcsBrowserFields(t *testing.T) {
	controlExpect := `Package: 
Version: 0.0.0
Architecture: amd64
Maintainer:  <>
Installed-Size: 0
Vcs-Git: https://github.com/xor-gate/debpkg.git
Vcs-Browser: https://github.com/xor-gate/debpkg
Description: 
`
	// Empty
	deb := New()
	defer deb.Close()

	// architecture is auto-set when empty, this makes sure it is always set to amd64
	deb.SetArchitecture("amd64")
	deb.SetVcsType(VcsTypeGit)
	deb.SetVcsURL("https://github.com/xor-gate/debpkg.git")
	deb.SetVcsBrowser("https://github.com/xor-gate/debpkg")
	control := deb.control.String(0)

	if control != controlExpect {
		t.Error("Unexpected control file")
		fmt.Printf("--- expected (len %d):\n'%s'\n--- got (len %d):\n'%s'---\n", len(controlExpect), controlExpect, len(control), control)
	}
}

// Test correct output of the control file when SetVersion* functions are called
// Only the mandatory fields are exported then, this behaviour is checked
func TestControlFileSetVersionMajorMinorPatch(t *testing.T) {
	// Empty
	deb := New()
	defer deb.Close()

	deb.SetName("foobar")
	deb.SetArchitecture("amd64")

	// Set major.minor.patch, leave full version string untouched
	deb.SetVersionMajor(1)
	deb.SetVersionMinor(2)
	deb.SetVersionPatch(3)

	controlExpect := `Package: foobar
Version: 1.2.3
Architecture: amd64
Maintainer:  <>
Installed-Size: 0
Description: 
`
	control := deb.control.String(0)

	if control != controlExpect {
		t.Error("Unexpected control file")
		fmt.Printf("--- expected (len %d):\n'%s'\n--- got (len %d):\n'%s'---\n", len(controlExpect), controlExpect, len(control), control)
	}

	// Set full version string, this will overwrite the set SetVersion{Major,Minor,Patch} string
	deb.SetVersion("7.8.9")
	control = deb.control.String(0)

	controlExpectFullVersion := `Package: foobar
Version: 7.8.9
Architecture: amd64
Maintainer:  <>
Installed-Size: 0
Description: 
`

	if control != controlExpectFullVersion {
		t.Error("Unexpected control file")
		fmt.Printf("--- expected (len %d):\n'%s'\n--- got (len %d):\n'%s'---\n", len(controlExpect), controlExpect, len(control), control)
	}
}

// Test correct output of control file when the mandatory DepPkg Set* functions are called
// This checks if the long description is formatted according to the debian policy
func TestControlFileLongDescriptionFormatting(t *testing.T) {
	controlExpect := `Package: debpkg
Version: 0.0.0
Architecture: amd64
Maintainer: Jerry Jacobs <foo@bar.com>
Installed-Size: 0
Homepage: https://github.com/xor-gate/debpkg
Description: Golang package for creating (gpg signed) debian packages
 **Features**
 
 * Create simple debian packages from files and folders
 * Add custom control files (preinst, postinst, prerm, postrm etcetera)
 * dpkg like tool with a subset of commands (--contents, --control, --extract, --info)
 * Create package from debpkg.yml specfile (like packager.io without cruft)
 * GPG sign package
 * GPG verify package`

	// User supplied very long description without leading spaces and no ending newline
	controlDescr := `**Features**

* Create simple debian packages from files and folders
* Add custom control files (preinst, postinst, prerm, postrm etcetera)
* dpkg like tool with a subset of commands (--contents, --control, --extract, --info)
* Create package from debpkg.yml specfile (like packager.io without cruft)
* GPG sign package
* GPG verify package`

	// Empty
	deb := New()
	defer deb.Close()

	deb.SetName("debpkg")
	deb.SetVersion("0.0.0")
	deb.SetMaintainer("Jerry Jacobs")
	deb.SetMaintainerEmail("foo@bar.com")
	deb.SetHomepage("https://github.com/xor-gate/debpkg")
	deb.SetShortDescription("Golang package for creating (gpg signed) debian packages")
	deb.SetDescription(controlDescr)
	// architecture is auto-set when empty, this makes sure it is always set to amd64
	deb.SetArchitecture("amd64")
	control := deb.control.String(0)

	if control != controlExpect {
		t.Error("Unexpected control file")
		fmt.Printf("--- expected (len %d):\n'%s'\n--- got (len %d):\n'%s'---\n", len(controlExpect), controlExpect, len(control), control)
	}
}

// Test correct output of a control file Installed-Size property
func TestControlInstalledSize(t *testing.T) {
	controlExpect1K := `Package: 
Version: 0.0.0
Architecture: amd64
Maintainer:  <>
Installed-Size: 1
Description: 
`
	// Empty
	deb := New()
	defer deb.Close()

	// architecture is auto-set when empty, this makes sure it is always set to amd64
	deb.SetArchitecture("amd64")
	control := deb.control.String(1024)

	if control != controlExpect1K {
		t.Error("Unexpected control file")
	}

	controlExpect2K := `Package: 
Version: 0.0.0
Architecture: amd64
Maintainer:  <>
Installed-Size: 2
Description: 
`

	// 1KByte + 1 byte
	control = deb.control.String(1025)
	if control != controlExpect2K {
		t.Error("Unexpected control file")
	}

	// 2KByte
	control = deb.control.String(2048)
	if control != controlExpect2K {
		t.Error("Unexpected control file")
	}
}