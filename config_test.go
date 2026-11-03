// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjey/debpkg/internal/test"
)

// TestExampleConfig verifies if the config example in the root is correctly loaded
func TestExampleConfig(t *testing.T) {
	const configFile = `name: debpkg
version: 7.6.5
architecture: all
maintainer: Deb Pkg
maintainer_email: deb@pkg.com
homepage: https://github.com/cjey/debpkg
section: devel
priority: standard
depends: lsb-release
recommends: nano
suggests: curl
conflicts: pico
provides: editor
replaces: vim
built_using: golang
description:
   short: This is a short description
   long: >
       Bla bla
       Bla Bla
       .
       Dusse
files:
  - file: LICENSE
    dest: {{.DATAROOTDIR}}/foobar/LICENSE
  - file: debpkg.go
  - file: debpkg_test.go
  - file: README.md
    dest: {{.DATAROOTDIR}}/foobar/README.md
  - dest: /bin/hello
    content: >
      #!/bin/bash
      echo "hello"
directories:
  - ./internal
emptydirs:
  - /var/cache/foobar
control_extra:
  postrm: Makefile
  prerm: Makefile
  postinst: Makefile
  preinst: Makefile
`
	cfg := test.WriteTempFile(t, "debpkg.yml", configFile)

	deb := New()
	defer deb.Close()

	assert.NoError(t, deb.Config(cfg))
	assert.Equal(t, "7.6.5", deb.control.info.version.full,
		"Unexpected deb.control.info.version.full")
	assert.Equal(t, "Deb Pkg", deb.control.info.maintainer,
		"Unexpected deb.control.info.maintainer")
	assert.Equal(t, "deb@pkg.com", deb.control.info.maintainerEmail,
		"Unexpected deb.control.info.maintainerEmail")
	assert.Equal(t, "https://github.com/cjey/debpkg", deb.control.info.homepage,
		"Unexpected deb.control.info.homepage")
	assert.Equal(t, "This is a short description", deb.control.info.descrShort,
		"Unexpected short description")
	assert.Equal(t, "golang", deb.control.info.builtUsing,
		"unexpected built using")
	assert.Equal(t, "devel", deb.control.info.section,
		"unexpected section")
	assert.Equal(t, PriorityStandard, deb.control.info.priority,
		"unexpected priority")

	assert.NoError(t, testWrite(t, deb))
}

func TestExampleConfigWithControlExtraContent(t *testing.T) {
	const configFile = `name: foo-bar
version: 1.2.3
architecture: amd64
maintainer: Mr. Foo Bar
maintainer_email: foo@bar.org
homepage: https://www.debian.org
section: net
priority: important
description:
  short: control extra content
control_extra:
  postrm: |
    #!/bin/bash
    echo "post rm!!"
  prerm: |
    #!/bin/bash
    echo "pre rm!!"
  postinst: |
    #!/bin/bash
    echo "post inst!!"
  preinst: "#!/bin/sh"
`
	cfg := test.WriteTempFile(t, "debpkg.yml", configFile)

	deb := New()
	defer deb.Close()

	assert.NoError(t, deb.Config(cfg))
	assert.Equal(t, "1.2.3", deb.control.info.version.full,
		"Unexpected deb.control.info.version.full")
	assert.Equal(t, "Mr. Foo Bar", deb.control.info.maintainer,
		"Unexpected deb.control.info.maintainer")
	assert.Equal(t, "foo@bar.org", deb.control.info.maintainerEmail,
		"Unexpected deb.control.info.maintainerEmail")
	assert.Equal(t, "https://www.debian.org", deb.control.info.homepage,
		"Unexpected deb.control.info.homepage")
	assert.Equal(t, "net", deb.control.info.section,
		"unexpected section")
	assert.Equal(t, PriorityImportant, deb.control.info.priority,
		"unexpected priority")

	f := test.TempFile(t)
	require.NoError(t, deb.Write(f))
	testReadWithNativeDpkg(t, f)
	control := readDeb(t, f).control
	// A single line with a shebang is a script too
	assert.Equal(t, "#!/bin/sh", string(control["preinst"].body))
	assert.Equal(t, "#!/bin/bash\necho \"post inst!!\"\n", string(control["postinst"].body))
	for _, name := range []string{"preinst", "postinst", "prerm", "postrm"} {
		assert.Equal(t, int64(0755), control[name].Mode, name)
	}
}

func TestExampleConfigWithConfigFile(t *testing.T) {
	const configFile = `name: bar-bar
version: 1.1.1
architecture: amd64
maintainer: Mr. Foo Bar
maintainer_email: foo@bar.org
homepage: https://www.debian.org
section: net
priority: important
description:
  short: config files
files:
  - dest: /etc/hello
    conffile: true
    content: |
      #!/bin/bash
      echo "hello"
  - dest: /my/awesome/makefile
    conffile: true
    file: Makefile
  - file: LICENSE
    conffile: true

`
	cfg := test.WriteTempFile(t, "debpkg.yml", configFile)

	deb := New()
	defer deb.Close()

	assert.NoError(t, deb.Config(cfg))
	assert.Equal(t, "1.1.1", deb.control.info.version.full,
		"Unexpected deb.control.info.version.full")
	// A file without dest is marked at its own path
	assert.Equal(t, []string{"/etc/hello", "/my/awesome/makefile", "/LICENSE"}, deb.control.conffiles)

	f := test.TempFile(t)
	require.NoError(t, deb.Write(f))
	testReadWithNativeDpkg(t, f)
	assert.Equal(t, "/etc/hello\n/my/awesome/makefile\n/LICENSE\n", string(readDeb(t, f).control["conffiles"].body))
}

// TestDefaultConfig verifies only the version, section and priority have defaults, nothing else
// is made up
func TestDefaultConfig(t *testing.T) {
	cfg := test.WriteTempFile(t, "debpkg.yml", "")

	deb := New()
	defer deb.Close()

	assert.NoError(t, deb.Config(cfg))
	assert.Equal(t, "0.1.0+dev", deb.control.info.version.full)
	assert.Equal(t, "misc", deb.control.info.section)
	assert.Equal(t, PriorityOptional, deb.control.info.priority)
	info := deb.control.info
	for _, v := range []string{info.name, info.architecture, info.maintainer,
		info.maintainerEmail, info.homepage, info.builtUsing, info.descrShort, info.descr} {
		assert.Empty(t, v)
	}
	assert.EqualError(t, deb.Write(test.TempFile(t)), "empty package name")
}

func TestNonExistingConfig(t *testing.T) {
	deb := New()
	defer deb.Close()

	assert.Error(t, deb.Config("/non/existent/config/file"))
}

func TestInvalidYAML(t *testing.T) {
	deb := New()
	defer deb.Close()

	const configFile = `name: debpkg
	foo: bar
	`
	assert.Error(t, deb.Config(test.WriteTempFile(t, "debpkg.yml", configFile)))
}

func TestInvalidTemplateVar(t *testing.T) {
	deb := New()
	defer deb.Close()

	const configFile = `name: debpkg
foo: {{.bar}}
	`
	assert.Error(t, deb.Config(test.WriteTempFile(t, "debpkg.yml", configFile)))
}

func TestConfigUnknownKey(t *testing.T) {
	deb := New()
	defer deb.Close()

	cfg := test.WriteTempFile(t, "debpkg.yml", "name: foo\nmaintainer_mail: foo@bar.com\n")
	assert.ErrorContains(t, deb.Config(cfg), "maintainer_mail")
}

func TestConfigControlExtraMissingFile(t *testing.T) {
	deb := New()
	defer deb.Close()

	cfg := test.WriteTempFile(t, "debpkg.yml", "control_extra:\n  postinst: /no/such/postinst\n")
	assert.ErrorContains(t, deb.Config(cfg), "add postinst")
}

func TestConfigFileError(t *testing.T) {
	for config, want := range map[string]string{
		"files:\n  - file: LICENSE\n    content: x\n    dest: /x\n": "has both 'file' and 'content'",
		"files:\n  - content: x\n":                                  "empty destination path",
		"files:\n  - dest: /x\n":                                    "either 'file' or 'content'",
		"files:\n  - file: LICENSE\n    dest: /opt/foo/\n":          "destination path /opt/foo/ ends with a slash",
	} {
		deb := New()
		assert.ErrorContains(t, deb.Config(test.WriteTempFile(t, "debpkg.yml", config)), want, config)
		deb.Close()
	}
}

// TestConfigConffileInDirectory verifies a file of the directories is marked as a conffile by
// listing it in files too
func TestConfigConffileInDirectory(t *testing.T) {
	const configFile = `name: conffile-in-directory
architecture: all
description:
  short: conffile in a directory
files:
  - file: internal/config/config.go
    conffile: true
directories:
  - internal
`
	deb := New()
	defer deb.Close()

	require.NoError(t, deb.Config(test.WriteTempFile(t, "debpkg.yml", configFile)))
	f := test.TempFile(t)
	require.NoError(t, deb.Write(f))
	testReadWithNativeDpkg(t, f)
	assert.Equal(t, "/internal/config/config.go\n", string(readDeb(t, f).control["conffiles"].body))
}

func TestConfigFieldsAndVcs(t *testing.T) {
	const configFile = `vcs_type: Git
vcs_url: https://github.com/cjey/debpkg.git
vcs_browser: https://github.com/cjey/debpkg
fields:
  Pre-Depends: dpkg (>= 1.19)
  Git-Commit: e9caee8
`
	deb := New()
	defer deb.Close()

	require.NoError(t, deb.Config(test.WriteTempFile(t, "debpkg.yml", configFile)))
	control := deb.control.String(0)
	assert.Contains(t, control, "\nVcs-Git: https://github.com/cjey/debpkg.git\nVcs-Browser: https://github.com/cjey/debpkg\n")
	assert.Contains(t, control, "\nPre-Depends: dpkg (>= 1.19)\nGit-Commit: e9caee8\nDescription: ")

	other := New()
	defer other.Close()
	assert.ErrorContains(t, other.Config(test.WriteTempFile(t, "debpkg.yml", "fields:\n  - Pre-Depends\n")),
		"fields must be a mapping")
}
