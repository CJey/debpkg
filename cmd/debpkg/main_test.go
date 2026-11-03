package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const config = `name: foo
version: 1.0
architecture: all
maintainer: Foo Bar
maintainer_email: foo@bar.com
description:
  short: debpkg command test
`

func writeConfig(t *testing.T, dir string) string {
	p := filepath.Join(dir, "debpkg.yml")
	require.NoError(t, os.WriteFile(p, []byte(config), 0644))
	return p
}

func TestRun(t *testing.T) {
	t.Setenv("DEBPKG_VERSION", "")
	dir := t.TempDir()
	out := filepath.Join(dir, "out.deb")
	var stdout bytes.Buffer
	require.NoError(t, run([]string{"-c", writeConfig(t, dir), "-o", out}, &stdout))
	assert.Equal(t, "debpkg: written: "+out+"\n", stdout.String())
	assert.FileExists(t, out)
}

func TestRunDefaultOutput(t *testing.T) {
	t.Setenv("DEBPKG_VERSION", "2.0")
	dir := t.TempDir()
	writeConfig(t, dir)
	t.Chdir(dir)
	var stdout bytes.Buffer
	require.NoError(t, run(nil, &stdout))
	assert.Equal(t, "debpkg: written: foo_2.0_all.deb\n", stdout.String())
	assert.FileExists(t, filepath.Join(dir, "foo_2.0_all.deb"))
}

func TestRunDefaultVersion(t *testing.T) {
	t.Setenv("DEBPKG_VERSION", "")
	dir := t.TempDir()
	cfg := filepath.Join(dir, "debpkg.yml")
	require.NoError(t, os.WriteFile(cfg, []byte(strings.Replace(config, "version: 1.0\n", "", 1)), 0644))
	t.Chdir(dir)
	var stdout bytes.Buffer
	require.NoError(t, run(nil, &stdout))
	assert.Equal(t, "debpkg: written: foo_0.1.0+dev_all.deb\n", stdout.String())
}

// TestRunToStdout verifies a package written to stdout, as with -o /dev/stdout, is left intact
func TestRunToStdout(t *testing.T) {
	t.Setenv("DEBPKG_VERSION", "")
	dir := t.TempDir()
	out := filepath.Join(dir, "out.deb")
	stdout, err := os.Create(out)
	require.NoError(t, err)
	defer stdout.Close()
	require.NoError(t, run([]string{"-c", writeConfig(t, dir), "-o", out}, stdout))
	b, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(b, []byte("!<arch>\ndebian-binary ")), "%q", b[:min(len(b), 32)])
}

// TestRunError verifies the intermediate files are removed on failure
func TestRunError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	assert.Error(t, run([]string{"-c", filepath.Join(tmp, "missing.yml")}, io.Discard))
	assert.EqualError(t, run([]string{"foo.yml"}, io.Discard),
		"unexpected argument foo.yml, the specfile is given by -c")

	cfg := filepath.Join(t.TempDir(), "debpkg.yml")
	require.NoError(t, os.WriteFile(cfg, []byte("version: 1.0\n"), 0644))
	assert.EqualError(t, run([]string{"-c", cfg}, io.Discard), "write package: empty package name")

	entries, err := os.ReadDir(tmp)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
