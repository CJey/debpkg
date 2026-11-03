// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"archive/tar"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjey/debpkg/internal/targzip"
)

func newData(t *testing.T) *data {
	t.Helper()
	tgz, err := targzip.NewTempFile(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { tgz.Remove() })
	return &data{tgz: tgz}
}

// archive completes the archive and returns its entries
func archive(t *testing.T, d *data) []tarEntry {
	t.Helper()
	require.NoError(t, d.tgz.Close())
	b, err := os.ReadFile(d.tgz.Name())
	require.NoError(t, err)
	return readTarGz(t, b)
}

func entryNames(entries []tarEntry) []string {
	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names
}

func TestDebPath(t *testing.T) {
	for in, want := range map[string]string{
		"":             "",
		".":            "",
		"/":            "",
		"/my/foo/":     "my/foo",
		"./a/../b":     "b",
		"a//b":         "a/b",
		"../x":         "x",
		"/usr/bin/foo": "usr/bin/foo",
	} {
		got, err := debPath(in)
		assert.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	_, err := debPath("a\nb")
	assert.Error(t, err)
}

func TestDataAddDirectory(t *testing.T) {
	d := newData(t)
	require.NoError(t, d.addDirectory("/my/foo/directory/"))
	require.NoError(t, d.addDirectory("my/foo"))
	entries := archive(t, d)
	assert.Equal(t, []string{"my", "my/foo", "my/foo/directory"}, entryNames(entries))
	for _, e := range entries {
		assert.Equal(t, byte(tar.TypeDir), e.Typeflag, e.Name)
		assert.Equal(t, int64(0755), e.Mode, e.Name)
		assert.Equal(t, "root", e.Uname, e.Name)
	}
	assert.Equal(t, uint64(3), d.size)
}

func TestDataAddDirectoryRoot(t *testing.T) {
	d := newData(t)
	for _, dir := range []string{"", ".", "/"} {
		assert.NoError(t, d.addDirectory(dir), dir)
	}
	assert.Empty(t, archive(t, d))
}

func TestDataAddDirectoryError(t *testing.T) {
	d := newData(t)
	require.NoError(t, d.tgz.Close())
	assert.Error(t, d.addDirectory("/doesnt/matter"))
}

func TestDataAddFileString(t *testing.T) {
	d := newData(t)
	require.NoError(t, d.addFileString("test", "/foo"))
	assert.Equal(t, "098f6bcd4621d373cade4e832627b4f6  foo\n", d.md5sums.String())
	entries := archive(t, d)
	require.Len(t, entries, 1)
	assert.Equal(t, "foo", entries[0].Name)
	assert.Equal(t, int64(0644), entries[0].Mode)
	assert.Equal(t, "test", string(entries[0].body))
}

func TestDataAddFileWithDest(t *testing.T) {
	src := filepath.Join(t.TempDir(), "hello")
	require.NoError(t, os.WriteFile(src, []byte("hello\n"), 0644))
	require.NoError(t, os.Chmod(src, 0755))
	d := newData(t)
	require.NoError(t, d.addFile(src, "/usr/bin/hello"))
	assert.Equal(t, "b1946ac92492d2347c6235b4d2611184  usr/bin/hello\n", d.md5sums.String())
	entries := archive(t, d)
	assert.Equal(t, []string{"usr", "usr/bin", "usr/bin/hello"}, entryNames(entries))
	assert.Equal(t, int64(0755), entries[2].Mode)
	assert.Equal(t, "hello\n", string(entries[2].body))
	assert.Equal(t, uint64(3), d.size)
}

// TestDataLongNames verifies names and link targets over 100 bytes or not in ASCII are kept
func TestDataLongNames(t *testing.T) {
	d := newData(t)
	long := "usr/share/" + strings.Repeat("d", 101) + "/中文文件名.txt"
	target := strings.Repeat("./", 60) + "target"
	require.NoError(t, d.addFileString("x", long))
	require.NoError(t, d.addSymlink("/usr/share/link", nil, target, time.Unix(1700000000, 0)))
	entries := archive(t, d)
	assert.Equal(t, []string{"usr", "usr/share", path.Dir(long), long, "usr/share/link"}, entryNames(entries))
	assert.Equal(t, target, entries[4].Linkname)
}

func TestDataAddFileError(t *testing.T) {
	d := newData(t)
	assert.Error(t, d.addFile("/no/such/file", "/foo"))
	assert.ErrorContains(t, d.addFile(t.TempDir(), "/foo"), "not a regular file")
	assert.Empty(t, d.md5sums.String())
	assert.Empty(t, archive(t, d))
}

func TestDataAddFileStringError(t *testing.T) {
	d := newData(t)
	require.NoError(t, d.tgz.Close())
	assert.Error(t, d.addFileString("test", "/foo"))
	assert.Empty(t, d.md5sums.String())
}

func TestDataConflicts(t *testing.T) {
	d := newData(t)
	require.NoError(t, d.addFileString("a", "/etc/foo"))
	assert.ErrorContains(t, d.addFileString("b", "etc/foo"), "already exists")
	assert.ErrorContains(t, d.addDirectory("/etc/foo"), "not a directory")
	assert.ErrorContains(t, d.addFileString("c", "/etc/foo/bar"), "not a directory")
	assert.ErrorContains(t, d.addFileString("d", ""), "empty destination path")
	assert.ErrorContains(t, d.addFileString("e", "/"), "empty destination path")
	assert.EqualError(t, d.addFileString("f", "/opt/foo/"), "destination path /opt/foo/ ends with a slash")
	assert.Equal(t, []string{"etc", "etc/foo"}, entryNames(archive(t, d)))
}

// TestDataAddTree verifies each entry of a nested tree is added once, links kept as links
func TestDataAddTree(t *testing.T) {
	base := t.TempDir()
	for name, content := range map[string]string{"tree/a/b/c.txt": "c", "tree/a/d.txt": "d", "tree/e.txt": "e"} {
		p := filepath.Join(base, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0644))
	}
	require.NoError(t, os.Symlink("a/d.txt", filepath.Join(base, "tree/link-file")))
	require.NoError(t, os.Symlink("a", filepath.Join(base, "tree/link-dir")))
	t.Chdir(base)

	d := newData(t)
	require.NoError(t, d.addTree("tree"))
	entries := archive(t, d)
	assert.Equal(t, []string{"tree", "tree/a", "tree/a/b", "tree/a/b/c.txt", "tree/a/d.txt", "tree/e.txt",
		"tree/link-dir", "tree/link-file"}, entryNames(entries))
	links := map[string]string{}
	for _, e := range entries {
		if e.Typeflag == tar.TypeSymlink {
			links[e.Name] = e.Linkname
		}
	}
	assert.Equal(t, map[string]string{"tree/link-dir": "a", "tree/link-file": "a/d.txt"}, links)
	assert.Equal(t, 3, strings.Count(d.md5sums.String(), "\n"), "links have no md5sum")
}

// TestDataSameSource verifies the same file or link from disk is added once at a path, while
// another file there is refused
func TestDataSameSource(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "tree"), 0755))
	for _, name := range []string{"tree/a.conf", "other.conf"} {
		require.NoError(t, os.WriteFile(filepath.Join(base, name), []byte(name), 0644))
	}
	require.NoError(t, os.Symlink("a.conf", filepath.Join(base, "tree/link")))
	t.Chdir(base)

	d := newData(t)
	require.NoError(t, d.addFile("tree/a.conf", "/tree/a.conf"))
	require.NoError(t, d.addTree("tree"))
	require.NoError(t, d.addTree("tree"))
	assert.ErrorContains(t, d.addFile("other.conf", "tree/a.conf"), "already exists")
	assert.ErrorContains(t, d.addFileString("x", "tree/a.conf"), "already exists")
	assert.Equal(t, []string{"tree", "tree/a.conf", "tree/link"}, entryNames(archive(t, d)))
	assert.Equal(t, 1, strings.Count(d.md5sums.String(), "\n"))
}

func TestDataAddTreeSymlinkedRoot(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "real/sub"), 0755))
	require.NoError(t, os.Symlink("real", filepath.Join(base, "link")))
	t.Chdir(base)

	d := newData(t)
	require.NoError(t, d.addTree("link"))
	assert.Equal(t, []string{"link", "link/sub"}, entryNames(archive(t, d)))
}

func TestDataAddTreeNotDirectory(t *testing.T) {
	d := newData(t)
	assert.ErrorContains(t, d.addTree("data.go"), "not a directory")
	assert.Error(t, d.addTree("/no/such/dir"))
}
