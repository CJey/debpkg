// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjey/debpkg/internal/test"
)

// setMandatory sets the fields Write requires
func setMandatory(deb *DebPkg, name string) {
	deb.SetName(name)
	deb.SetVersion("1.0")
	deb.SetArchitecture("all")
	deb.SetMaintainer("Foo Bar")
	deb.SetMaintainerEmail("foo@bar.com")
	deb.SetShortDescription("debpkg test package")
}

// newTestDeb returns a package with the mandatory fields set, closed when the test ends
func newTestDeb(t *testing.T, name string) *DebPkg {
	deb := New()
	t.Cleanup(func() { deb.Close() })
	setMandatory(deb, name)
	return deb
}

// testWrite writes the package to a temporary file and checks it with dpkg-deb and lintian when available
func testWrite(t *testing.T, deb *DebPkg) error {
	t.Helper()
	f := test.TempFile(t)
	if err := deb.Write(f); err != nil {
		return err
	}
	testReadWithNativeDpkg(t, f)
	testReadWithLintian(t, f)
	return nil
}

// testReadWithNativeDpkg parses the control file and lists the data with dpkg-deb when present
func testReadWithNativeDpkg(t *testing.T, filename string) {
	t.Helper()
	dpkgDeb, err := exec.LookPath("dpkg-deb")
	if err != nil {
		return
	}
	for _, action := range []string{"--show", "--contents"} {
		out, err := exec.Command(dpkgDeb, action, filename).CombinedOutput()
		assert.NoError(t, err, "dpkg-deb %s: %s", action, out)
	}
}

// testReadWithLintian only checks lintian can process the package, as the test packages lack a
// copyright file and the like, which lintian reports as errors
func testReadWithLintian(t *testing.T, filename string) {
	t.Helper()
	lintian, err := exec.LookPath("lintian")
	if err != nil {
		return
	}
	out, err := exec.Command(lintian, "--fail-on", "none", filename).CombinedOutput()
	assert.NoError(t, err, "lintian: %s", out)
}

// testInstall installs the package into a scratch root with fakeroot and dpkg when present,
// and returns the root, empty when skipped. The package must have no maintainer scripts,
// which would run on the host.
func testInstall(t *testing.T, filename string) string {
	t.Helper()
	fakeroot, err := exec.LookPath("fakeroot")
	if err != nil {
		t.Log("fakeroot not found, install skipped")
		return ""
	}
	dpkg, err := exec.LookPath("dpkg")
	if err != nil {
		t.Log("dpkg not found, install skipped")
		return ""
	}
	root := t.TempDir()
	admin := filepath.Join(root, "var/lib/dpkg")
	for _, dir := range []string{"info", "updates", "triggers"} {
		require.NoError(t, os.MkdirAll(filepath.Join(admin, dir), 0755))
	}
	for _, file := range []string{"status", "available"} {
		require.NoError(t, os.WriteFile(filepath.Join(admin, file), nil, 0644))
	}
	out, err := exec.Command(fakeroot, dpkg, "--force-not-root", "--force-bad-path", "--root="+root,
		"--admindir="+admin, "--log=/dev/null", "--install", filename).CombinedOutput()
	require.NoError(t, err, "dpkg --install: %s", out)
	return root
}

type tarEntry struct {
	*tar.Header
	body []byte
}

type debFile struct {
	members []string // ar members in order
	raw     map[string][]byte
	mtimes  map[string]int64
	control map[string]tarEntry
	data    []tarEntry
}

// readDeb parses a package, checking the ar structure on the way
func readDeb(t *testing.T, filename string) *debFile {
	t.Helper()
	b, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(b, []byte("!<arch>\n")), "ar magic missing")
	d := &debFile{raw: map[string][]byte{}, mtimes: map[string]int64{}, control: map[string]tarEntry{}}
	for b = b[8:]; len(b) > 0; {
		require.GreaterOrEqual(t, len(b), 60, "truncated ar header")
		require.Equal(t, "`\n", string(b[58:60]), "bad ar header end")
		name := strings.TrimRight(string(b[:16]), " ")
		mtime, err := strconv.ParseInt(strings.TrimRight(string(b[16:28]), " "), 10, 64)
		require.NoError(t, err)
		d.mtimes[name] = mtime
		size, err := strconv.Atoi(strings.TrimRight(string(b[48:58]), " "))
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(b), 60+size+size%2, "truncated ar member %s", name)
		if size%2 == 1 {
			require.Equal(t, byte('\n'), b[60+size], "bad padding of ar member %s", name)
		}
		d.members = append(d.members, name)
		d.raw[name] = b[60 : 60+size]
		b = b[60+size+size%2:]
	}
	for _, e := range readTarGz(t, d.raw["control.tar.gz"]) {
		d.control[e.Name] = e
	}
	d.data = readTarGz(t, d.raw["data.tar.gz"])
	return d
}

func readTarGz(t *testing.T, b []byte) []tarEntry {
	t.Helper()
	gr, err := gzip.NewReader(bytes.NewReader(b))
	require.NoError(t, err)
	tr := tar.NewReader(gr)
	var entries []tarEntry
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return entries
		}
		require.NoError(t, err)
		assert.Equal(t, tar.FormatGNU, hdr.Format, "dpkg cannot read %s in another format", hdr.Name)
		body, err := io.ReadAll(tr)
		require.NoError(t, err)
		entries = append(entries, tarEntry{hdr, body})
	}
}
