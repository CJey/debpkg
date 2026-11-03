// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjey/debpkg/internal/test"
)

// Test correct output of a empty control file when no DepPkg Set* functions are called
// Only the mandatory fields are exported then, this behaviour is checked
func TestControlFileEmpty(t *testing.T) {
	controlExpect := "Package: \n" +
		"Version: 0.0.0\n" +
		"Architecture: amd64\n" +
		"Installed-Size: 0\n" +
		"Description: \n"
	// Empty
	deb := New()
	defer deb.Close()

	deb.SetArchitecture("amd64")

	assert.Equal(t, controlExpect, deb.control.String(0))
}

// Test correct output of a control file when SetVcs* functions are called
// Only the mandatory fields are exported then, this behaviour is checked
func TestControlFileVcsAndVcsBrowserFields(t *testing.T) {
	controlExpect := "Package: \n" +
		"Version: 0.0.0\n" +
		"Architecture: amd64\n" +
		"Installed-Size: 0\n" +
		"Vcs-Git: https://github.com/cjey/debpkg.git\n" +
		"Vcs-Browser: https://github.com/cjey/debpkg\n" +
		"Description: \n"
	// Empty
	deb := New()
	defer deb.Close()

	deb.SetArchitecture("amd64")
	deb.SetVcsType(VcsTypeGit)
	deb.SetVcsURL("https://github.com/cjey/debpkg.git")
	deb.SetVcsBrowser("https://github.com/cjey/debpkg")

	assert.Equal(t, controlExpect, deb.control.String(0))
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

	controlExpect := "Package: foobar\n" +
		"Version: 1.2.3\n" +
		"Architecture: amd64\n" +
		"Installed-Size: 0\n" +
		"Description: \n"
	assert.Equal(t, controlExpect, deb.control.String(0))

	// Set full version string, this will overwrite the set SetVersion{Major,Minor,Patch} string
	deb.SetVersion("7.8.9")

	controlExpectFullVersion := "Package: foobar\n" +
		"Version: 7.8.9\n" +
		"Architecture: amd64\n" +
		"Installed-Size: 0\n" +
		"Description: \n"

	assert.Equal(t, controlExpectFullVersion, deb.control.String(0))
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
 .
 * Create simple debian packages from files and folders
 * Add custom control files (preinst, postinst, prerm, postrm etcetera)
 * dpkg like tool with a subset of commands (--contents, --control, --extract, --info)
 * Create package from debpkg.yml specfile (like packager.io without cruft)
 * GPG sign package
 * GPG verify package
`

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
	deb.SetArchitecture("amd64")

	assert.Equal(t, controlExpect, deb.control.String(0))
}

func TestControlFileDescription(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"", ""},
		{"one line", " one line\n"},
		{"para one\n\npara two\n", " para one\n .\n para two\n"},
		{"\n\nblank lines around\n\n\n", " blank lines around\n"},
		{"dos\r\nline endings\r\n", " dos\n line endings\n"},
		{"whitespace only\n \t\nline", " whitespace only\n .\n line\n"},
		{"trailing spaces  \n  verbatim", " trailing spaces\n   verbatim\n"},
		{"page one\n\f\npage two\v", " page one\n .\n page two\n"},
	} {
		var b strings.Builder
		writeDescription(&b, tc.in)
		assert.Equal(t, tc.want, b.String(), "%q", tc.in)
	}
}

func TestControlFileMaintainer(t *testing.T) {
	deb := New()
	defer deb.Close()

	for _, tc := range []struct{ name, email, want string }{
		{"Foo Bar", "foo@bar.com", "Foo Bar <foo@bar.com>"},
		{"Foo Bar", "", "Foo Bar"},
		{"", "foo@bar.com", "<foo@bar.com>"},
	} {
		deb.SetMaintainer(tc.name)
		deb.SetMaintainerEmail(tc.email)
		assert.Contains(t, deb.control.String(0), "\nMaintainer: "+tc.want+"\n")
	}
	deb.SetMaintainer("")
	deb.SetMaintainerEmail("")
	assert.NotContains(t, deb.control.String(0), "Maintainer")
}

// TestControlFileValidation verifies Write refuses what dpkg refuses
func TestControlFileValidation(t *testing.T) {
	for _, tc := range []struct {
		set  func(deb *DebPkg)
		want string
	}{
		{func(deb *DebPkg) { deb.SetName("foo bar") }, `invalid package name "foo bar"`},
		{func(deb *DebPkg) { deb.SetName("-foo") }, `invalid package name "-foo"`},
		{func(deb *DebPkg) { deb.SetVersion("v1.0") }, "does not start with a digit"},
		{func(deb *DebPkg) { deb.SetVersion("1.0_beta") }, "invalid character in upstream version"},
		{func(deb *DebPkg) { deb.SetVersion("a:1.0") }, "epoch is not a number"},
		{func(deb *DebPkg) { deb.SetVersion("1.0-") }, "empty revision"},
		{func(deb *DebPkg) { deb.SetVersion("1:") }, "empty upstream version"},
		{func(deb *DebPkg) { deb.SetVersion("1.0-1_2") }, "invalid character in revision"},
		{func(deb *DebPkg) { deb.SetArchitecture("") }, "empty architecture"},
		{func(deb *DebPkg) { deb.SetArchitecture("amd 64") }, `invalid architecture "amd 64"`},
		{func(deb *DebPkg) { deb.SetArchitecture("any") }, "architecture any is not valid for a binary package"},
		{func(deb *DebPkg) { deb.SetArchitecture("source") }, "architecture source is not valid for a binary package"},
		{func(deb *DebPkg) { deb.SetArchitecture("linux-any") }, "architecture linux-any is not valid for a binary package"},
		{func(deb *DebPkg) { deb.SetArchitecture("any-amd64") }, "architecture any-amd64 is not valid for a binary package"},
		{func(deb *DebPkg) { deb.SetVcsType("Foo Bar"); deb.SetVcsURL("x") }, `invalid field name "Vcs-Foo Bar"`},
		{func(deb *DebPkg) { deb.SetVcsType("Git: x\nEssential"); deb.SetVcsURL("yes") }, "invalid field name"},
		{func(deb *DebPkg) { deb.SetShortDescription("") }, "empty short description"},
		{func(deb *DebPkg) { deb.SetShortDescription("two\nlines") }, "value of field Description is not a single line of text"},
		{func(deb *DebPkg) { deb.SetDepends("a\nEssential: yes") }, "value of field Depends is not a single line of text"},
		{func(deb *DebPkg) { deb.SetDescription("a\x1aEssential: yes") }, "extended description contains ^Z"},
	} {
		deb := newTestDeb(t, "validation")
		tc.set(deb)
		assert.ErrorContains(t, deb.Write(test.TempFile(t)), tc.want)
	}
}

func TestControlFileValidValues(t *testing.T) {
	for _, v := range []string{"1.0", "1:2.3-4", "2.0~rc1+git20261101", "0.1.0+dev", "1.2.3-1ubuntu1", "1.0-1-2", "1:2:3"} {
		assert.NoError(t, validVersion(v), v)
	}
	// dpkg accepts upper case, turned into lower case
	for _, name := range []string{"foo", "Foo", "lib++", "a.b_c", "0ad"} {
		assert.NoError(t, validPackageName(name), name)
	}
	for _, arch := range []string{"all", "amd64", "hurd-i386", "musl-linux-arm64"} {
		assert.NoError(t, validArchitecture(arch), arch)
	}
}

func TestControlFileExtraString(t *testing.T) {
	deb := newTestDeb(t, "debpkg-control-file-extra-string")
	deb.SetDescription("bla bla")

	scripts := []string{"preinst", "postinst", "prerm", "postrm"}
	for _, name := range scripts {
		script := "#!/bin/sh\r\necho \"" + name + ": hello world from debpkg!\"\r\n"
		require.NoError(t, deb.AddControlExtraString(name, script))
	}
	require.NoError(t, deb.AddControlExtraString("triggers", "interest-noawait /usr/share/foo\n"))

	f := test.TempFile(t)
	require.NoError(t, deb.Write(f))
	testReadWithNativeDpkg(t, f)
	control := readDeb(t, f).control
	for _, name := range scripts {
		assert.Equal(t, int64(0755), control[name].Mode, name)
		assert.Equal(t, "#!/bin/sh\necho \""+name+": hello world from debpkg!\"\n", string(control[name].body))
	}
	assert.Equal(t, int64(0644), control["triggers"].Mode)
	assert.Equal(t, int64(0644), control["control"].Mode)
	// Nothing to list
	assert.NotContains(t, control, "md5sums")
	assert.NotContains(t, control, "conffiles")
}

func TestControlFileExtra(t *testing.T) {
	deb := newTestDeb(t, "debpkg-control-file-extra")
	script := test.WriteTempFile(t, "script.sh", "#!/bin/sh\necho \"hello world from debpkg\"\n")
	for _, name := range []string{"preinst", "postinst", "prerm", "postrm"} {
		require.NoError(t, deb.AddControlExtra(name, script))
	}
	assert.NoError(t, testWrite(t, deb))
}

func TestControlFileExtraError(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"", "invalid control file name"},
		{"..", "invalid control file name"},
		{"a/b", "invalid control file name"},
		{"post inst", "invalid control file name"},
		{"control", "generated by debpkg"},
		{"md5sums", "generated by debpkg"},
	} {
		deb := newTestDeb(t, "extra")
		assert.ErrorContains(t, deb.AddControlExtraString(tc.name, "x"), tc.want, tc.name)
	}

	deb := newTestDeb(t, "extra")
	require.NoError(t, deb.AddControlExtraString("postinst", "#!/bin/sh\n"))
	assert.ErrorContains(t, deb.AddControlExtraString("postinst", "#!/bin/sh\n"), "already added")

	deb = newTestDeb(t, "extra")
	err := deb.AddControlExtra("postinst", "/no/such/file")
	assert.Error(t, err)
	assert.Equal(t, err, deb.Write(test.TempFile(t)), "the error is kept")
}

func TestConffiles(t *testing.T) {
	deb := newTestDeb(t, "conffiles")
	require.NoError(t, deb.AddFileString("a", "/etc/foo.conf"))
	require.NoError(t, deb.AddFileString("b", "etc/bar.conf"))
	require.NoError(t, deb.MarkConfigFile("etc/foo.conf"))
	require.NoError(t, deb.MarkConfigFile("/etc/foo.conf"))
	require.NoError(t, deb.MarkConfigFile("/etc/./bar.conf"))
	assert.Equal(t, []string{"/etc/foo.conf", "/etc/bar.conf"}, deb.control.conffiles)

	f := test.TempFile(t)
	require.NoError(t, deb.Write(f))
	conffiles := readDeb(t, f).control["conffiles"]
	assert.Equal(t, "/etc/foo.conf\n/etc/bar.conf\n", string(conffiles.body))
	assert.Equal(t, int64(0644), conffiles.Mode)
}

func TestConffilesError(t *testing.T) {
	for p, want := range map[string]string{
		"/":                                "empty config file path",
		"/etc/x ":                          "ends with whitespace",
		"/etc/x\t":                         "ends with whitespace",
		"/etc/" + strings.Repeat("x", 992): "longer than the 996 bytes",
	} {
		assert.ErrorContains(t, newTestDeb(t, "conffiles").MarkConfigFile(p), want, "%q", p)
	}
	assert.NoError(t, newTestDeb(t, "conffiles").MarkConfigFile("/etc/"+strings.Repeat("x", 991)))

	for _, tc := range []struct {
		setup func(deb *DebPkg) error
		want  string
	}{
		{
			func(deb *DebPkg) error { return deb.MarkConfigFile("/etc/missing.conf") },
			"conffile /etc/missing.conf is not in the package",
		},
		{
			func(deb *DebPkg) error {
				if err := deb.AddEmptyDirectory("/etc/dir"); err != nil {
					return err
				}
				return deb.MarkConfigFile("/etc/dir")
			},
			"conffile /etc/dir is not a regular file",
		},
		{
			func(deb *DebPkg) error {
				if err := deb.AddFileString("x", "/etc/x"); err != nil {
					return err
				}
				if err := deb.AddControlExtraString("conffiles", "/etc/x\n"); err != nil {
					return err
				}
				return deb.MarkConfigFile("/etc/x")
			},
			"conffiles are set both by MarkConfigFile and AddControlExtra",
		},
	} {
		deb := newTestDeb(t, "conffiles")
		require.NoError(t, tc.setup(deb))
		assert.EqualError(t, deb.Write(test.TempFile(t)), tc.want)
	}
}

// Test SetField: written after the standard fields in the order set, replaced or removed by a
// case-insensitive name, the fields of the other Set* functions and multiline values refused
func TestControlFileSetField(t *testing.T) {
	controlExpect := "Package: \n" +
		"Version: 0.0.0\n" +
		"Architecture: amd64\n" +
		"Installed-Size: 0\n" +
		"Depends: ally\n" +
		"Git-Commit: e9caee8\n" +
		"Pre-Depends: dpkg (>= 1.19)\n" +
		"Build-Time: 2026-10-28T14:22:16+08:00\n" +
		"Description: \n"
	deb := New()
	defer deb.Close()

	deb.SetArchitecture("amd64")
	deb.SetDepends("ally")
	assert.Nil(t, deb.SetField("Git-Commit", "0000000"))
	assert.Nil(t, deb.SetField("Pre-Depends", "dpkg (>= 1.19)"))
	assert.Nil(t, deb.SetField("Git-Dirty", "no"))
	assert.Nil(t, deb.SetField("Build-Time", " 2026-10-28T14:22:16+08:00\t"))
	assert.Nil(t, deb.SetField("git-commit", "e9caee8"))
	assert.Nil(t, deb.SetField("Git-Dirty", " "))
	assert.Equal(t, controlExpect, deb.control.String(0))

	// A fresh package each, as the first error is kept
	for _, name := range []string{"", "#Note", "-Note", "Git Commit", "Git:Commit", "Gitä", "version", "VCS-Git",
		"Status", "conffiles", "Triggers-Pending"} {
		deb := New()
		assert.NotNil(t, deb.SetField(name, "x"), name)
		deb.Close()
	}
	for _, value := range []string{"a\nb", "a\rb", "a\x00b"} {
		deb := New()
		assert.NotNil(t, deb.SetField("Note", value), value)
		deb.Close()
	}
}
