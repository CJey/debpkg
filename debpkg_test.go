// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjey/debpkg/internal/test"
)

// TestAddDirectory verifies adding a single directory recursive to the package
func TestAddDirectory(t *testing.T) {
	deb := newTestDeb(t, "debpkg-test-add-directory")
	assert.NoError(t, deb.AddDirectory("internal"))
	assert.NoError(t, testWrite(t, deb))
}

// TestWrite verifies Write works as expected with adding just one datafile
func TestWrite(t *testing.T) {
	deb := New()
	defer deb.Close()

	deb.SetName("debpkg-test")
	deb.SetArchitecture("all")
	deb.SetVersion("0.0.1")
	deb.SetMaintainer("Foo Bar")
	deb.SetMaintainerEmail("foo@bar.com")
	deb.SetHomepage("https://foobar.com")
	deb.SetShortDescription("some awesome foobar pkg")
	deb.SetDescription("very very very very long description")

	// Set version control system info for control file
	deb.SetVcsType(VcsTypeGit)
	deb.SetVcsURL("https://github.com/xor-gate/secdl")
	deb.SetVcsBrowser("https://github.com/xor-gate/secdl")

	assert.NoError(t, deb.AddFile("debpkg.go"))
	assert.NoError(t, deb.AddFile("debpkg_test.go", "/foo/awesome/test.go"))
	assert.NoError(t, deb.AddFileString("this is a real file", "/real/file.txt"))

	assert.NoError(t, testWrite(t, deb))

	// Try to Write again on implicit closed package
	assert.Equal(t, ErrClosed, testWrite(t, deb))
}

// TestWriteError tests if the Write fails with the correct errors
func TestWriteError(t *testing.T) {
	deb := New()
	defer deb.Close()
	assert.EqualError(t, deb.Write(""), "empty package name")

	// The error is kept, even with its cause gone
	deb.SetName("pkg")
	assert.EqualError(t, deb.Write(""), "empty package name")
}

// ExampleDebPkg_Write demonstrates generating a simple package
func ExampleDebPkg_Write() {
	dir, err := os.MkdirTemp("", "debpkg-example")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	deb := New()
	defer deb.Close()

	deb.SetName("foobar")
	deb.SetVersion("1.2.3")
	deb.SetArchitecture("amd64")
	deb.SetMaintainer("Foo Bar")
	deb.SetMaintainerEmail("foo@bar.com")
	deb.SetHomepage("http://foobar.com")

	deb.SetShortDescription("Minimal foo bar package")
	deb.SetDescription("Foo bar package doesn't do anything")

	deb.AddFile("debpkg.go", "/usr/share/foobar/debpkg.go")
	fmt.Println(deb.GetFilename(), deb.Write(filepath.Join(dir, deb.GetFilename())))
	// Output: foobar_1.2.3_amd64.deb <nil>
}

func TestGetFilename(t *testing.T) {
	deb := New()
	defer deb.Close()

	deb.SetName("foo")
	deb.SetArchitecture("amd64")
	deb.SetVersionMajor(1)
	deb.SetVersionMinor(2)
	deb.SetVersionPatch(3)
	assert.Equal(t, "foo_1.2.3_amd64.deb", deb.GetFilename())

	deb.SetVersion("1.33.7")
	assert.Equal(t, "foo_1.33.7_amd64.deb", deb.GetFilename())

	// The epoch is left out, like dpkg-name does
	deb.SetVersion("2:1.33.7-1")
	assert.Equal(t, "foo_1.33.7-1_amd64.deb", deb.GetFilename())
}

func TestGetArchitecture(t *testing.T) {
	for _, tc := range []struct{ goarch, goarm, want string }{
		{"386", "", "i386"},
		{"amd64", "", "amd64"},
		{"arm64", "", "arm64"},
		{"arm", "", "armhf"},
		{"arm", "7", "armhf"},
		{"arm", "6", "armhf"},
		{"arm", "6,softfloat", "armhf"},
		{"arm", "5", "armel"},
		{"ppc64le", "", "ppc64el"},
		{"mipsle", "", "mipsel"},
		{"mips64le", "", "mips64el"},
		{"riscv64", "", "riscv64"},
	} {
		t.Setenv("GOARCH", tc.goarch)
		t.Setenv("GOARM", tc.goarm)
		assert.Equal(t, tc.want, GetArchitecture(), "GOARCH=%s GOARM=%s", tc.goarch, tc.goarm)
	}

	t.Setenv("GOARCH", "")
	if runtime.GOARCH == "amd64" {
		assert.Equal(t, "amd64", GetArchitecture())
	}
}

// TestNewFailure verifies a package whose intermediate files cannot be created fails without panicking
func TestNewFailure(t *testing.T) {
	deb := New(filepath.Join(t.TempDir(), "missing"))
	assert.ErrorIs(t, deb.AddFileString("x", "/x"), ErrIO)
	assert.ErrorIs(t, deb.AddControlExtraString("postinst", "#!/bin/sh\n"), ErrIO)
	assert.ErrorIs(t, deb.MarkConfigFile("/x"), ErrIO)
	assert.ErrorIs(t, deb.WriteSigned("", test.OpenPGPIdentity(t)), ErrIO)
	assert.ErrorIs(t, deb.Write(""), ErrIO)
	assert.NoError(t, deb.Close())
}

// TestStickyError verifies a package is not written after an error was ignored
func TestStickyError(t *testing.T) {
	deb := newTestDeb(t, "sticky")
	err := deb.AddFile("/no/such/file")
	require.Error(t, err)
	assert.Equal(t, err, deb.AddFileString("x", "/x"))
	assert.Equal(t, err, deb.WriteSigned(test.TempFile(t), test.OpenPGPIdentity(t)))
	assert.Equal(t, err, deb.Write(test.TempFile(t)))
}

func TestTempFilesRemoved(t *testing.T) {
	dir := t.TempDir()
	count := func() int {
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		return len(entries)
	}

	deb := New(dir)
	assert.Equal(t, 2, count())
	assert.NoError(t, deb.Close())
	assert.Equal(t, 0, count())
	assert.Equal(t, ErrClosed, deb.Close())

	deb = New(dir)
	setMandatory(deb, "tempfiles")
	assert.NoError(t, deb.Write(test.TempFile(t)))
	assert.Equal(t, 0, count(), "after Write")

	deb = New(dir)
	assert.Error(t, deb.Write(test.TempFile(t)))
	assert.Equal(t, 0, count(), "after a failed Write")
}

// TestRelativeTempDir verifies the intermediate files survive a change of the working directory
func TestRelativeTempDir(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(base, "tmp"), 0755))
	t.Chdir(base)
	deb := New("tmp")
	setMandatory(deb, "relative-temp-dir")
	t.Chdir(t.TempDir())
	require.NoError(t, deb.Write(test.TempFile(t)))
	entries, err := os.ReadDir(filepath.Join(base, "tmp"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestInstalledSize(t *testing.T) {
	deb := newTestDeb(t, "installed-size")
	require.NoError(t, deb.AddFileString(strings.Repeat("x", 1025), "/a/b")) // 2 KiB, directory a 1 KiB
	require.NoError(t, deb.AddFileString(strings.Repeat("x", 1024), "/c"))   // 1 KiB
	require.NoError(t, deb.AddFileString("", "/empty"))
	f := test.TempFile(t)
	require.NoError(t, deb.Write(f))
	assert.Contains(t, string(readDeb(t, f).control["control"].body), "\nInstalled-Size: 4\n")
}

func TestInvalidSourceDateEpoch(t *testing.T) {
	for _, v := range []string{"yesterday", "-1", "1.5"} {
		t.Setenv("SOURCE_DATE_EPOCH", v)
		deb := New()
		assert.ErrorContains(t, deb.Write(""), "invalid SOURCE_DATE_EPOCH", v)
		assert.NoError(t, deb.Close())
	}
}

func TestReproducible(t *testing.T) {
	const epoch = 1700000000
	t.Setenv("SOURCE_DATE_EPOCH", strconv.Itoa(epoch))
	src := test.WriteTempFile(t, "file", "content\n") // modified after the epoch
	build := func(sign bool) *debFile {
		deb := newTestDeb(t, "reproducible")
		require.NoError(t, deb.AddFile(src, "/usr/share/reproducible/file"))
		require.NoError(t, deb.AddFileString("x\n", "/etc/reproducible.conf"))
		require.NoError(t, deb.AddControlExtraString("postinst", "#!/bin/sh\n"))
		f := test.TempFile(t)
		if sign {
			require.NoError(t, deb.WriteSigned(f, test.OpenPGPIdentity(t)))
		} else {
			require.NoError(t, deb.Write(f))
		}
		return readDeb(t, f)
	}

	first := build(false)
	for name, mtime := range first.mtimes {
		assert.Equal(t, int64(epoch), mtime, name)
	}
	for name, e := range first.control {
		assert.Equal(t, int64(epoch), e.ModTime.Unix(), name)
	}
	for _, e := range first.data {
		assert.Equal(t, int64(epoch), e.ModTime.Unix(), e.Name)
	}
	assert.Equal(t, first.raw, build(false).raw)

	// Only the signature differs between signed packages
	for range 2 {
		signed := build(true)
		for _, name := range first.members {
			assert.Equal(t, first.raw[name], signed.raw[name], name)
		}
	}
}

// TestInstallWithDpkg installs a package with directories, links, a conffile, and names over
// 100 bytes or not in ASCII
func TestInstallWithDpkg(t *testing.T) {
	base := t.TempDir()
	long := "tree/" + strings.Repeat("d", 101) + "/中文文件名.txt"
	longTarget := strings.Repeat("./", 60) + "a/d.txt"
	for name, content := range map[string]string{"tree/a/b/c.txt": "c\n", "tree/a/d.txt": "d\n", long: "long\n"} {
		p := filepath.Join(base, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0644))
	}
	require.NoError(t, os.Symlink("a/d.txt", filepath.Join(base, "tree/link")))
	require.NoError(t, os.Symlink(longTarget, filepath.Join(base, "tree/long-link")))
	t.Chdir(base)

	deb := newTestDeb(t, "debpkg-install")
	deb.SetDescription("First paragraph.\n\nSecond paragraph,\n  verbatim line.\n\f\nNext page.")
	require.NoError(t, deb.AddDirectory("tree"))
	require.NoError(t, deb.AddFileString("key=value\n", "/etc/debpkg-install.conf"))
	require.NoError(t, deb.MarkConfigFile("etc/debpkg-install.conf"))
	require.NoError(t, deb.AddEmptyDirectory("/var/cache/debpkg-install"))
	f := test.TempFile(t)
	require.NoError(t, deb.Write(f))
	testReadWithNativeDpkg(t, f)

	root := testInstall(t, f)
	if root == "" {
		return
	}
	b, err := os.ReadFile(filepath.Join(root, "tree/a/b/c.txt"))
	require.NoError(t, err)
	assert.Equal(t, "c\n", string(b))
	target, err := os.Readlink(filepath.Join(root, "tree/link"))
	require.NoError(t, err)
	assert.Equal(t, "a/d.txt", target)
	b, err = os.ReadFile(filepath.Join(root, long))
	require.NoError(t, err)
	assert.Equal(t, "long\n", string(b))
	target, err = os.Readlink(filepath.Join(root, "tree/long-link"))
	require.NoError(t, err)
	assert.Equal(t, longTarget, target)
	assert.DirExists(t, filepath.Join(root, "var/cache/debpkg-install"))
	status, err := os.ReadFile(filepath.Join(root, "var/lib/dpkg/status"))
	require.NoError(t, err)
	assert.Contains(t, string(status), "Conffiles:\n /etc/debpkg-install.conf ")
}
