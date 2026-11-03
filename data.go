// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/cjey/debpkg/internal/targzip"
)

type entryKind uint8

const (
	kindDir entryKind = iota + 1
	kindFile
	kindSymlink
)

type data struct {
	tgz     *targzip.TarGzip
	date    sourceDate
	entries map[string]entryKind
	sources map[string]fs.FileInfo // files and links taken from disk
	md5sums strings.Builder
	size    uint64 // Installed-Size in KiB, estimated like dpkg-gencontrol
}

// debPath normalizes p to a path inside the archive: slash separated, relative to the root
// and without "." or ".." elements. The root itself is "".
func debPath(p string) (string, error) {
	p = filepath.ToSlash(p[len(filepath.VolumeName(p)):])
	p = strings.TrimPrefix(path.Clean("/"+p), "/")
	if strings.ContainsAny(p, "\n\r") {
		return "", fmt.Errorf("path %q contains a line break", p)
	}
	return p, nil
}

func (d *data) addDirectory(dir string) error {
	p, err := debPath(dir)
	if err != nil || p == "" {
		return err
	}
	switch d.entries[p] {
	case kindDir:
		return nil
	case 0:
	default:
		return fmt.Errorf("%s already exists in the package and is not a directory", p)
	}
	if err := d.addDirectory(path.Dir(p)); err != nil {
		return err
	}
	if err := d.tgz.AddDirectory(p, d.date.now()); err != nil {
		return err
	}
	d.setEntry(p, kindDir, nil)
	d.size++
	return nil
}

// reserve checks that dest can be added as a new file or link, and adds its parent directories.
// It returns "" when dest already holds src, the same file or link from disk, as a specfile adds
// a file of its directories again to mark it as a conffile.
func (d *data) reserve(dest string, src fs.FileInfo) (string, error) {
	p, err := debPath(dest)
	if err != nil {
		return "", err
	}
	switch {
	case p == "":
		return "", errors.New("empty destination path")
	case strings.HasSuffix(filepath.ToSlash(dest), "/"):
		// The file would replace the directory, which dpkg deletes with its content unless a package owns it
		return "", fmt.Errorf("destination path %s ends with a slash", dest)
	case src != nil && d.sources[p] != nil && os.SameFile(d.sources[p], src):
		return "", nil
	case d.entries[p] != 0:
		return "", fmt.Errorf("%s already exists in the package", p)
	}
	return p, d.addDirectory(path.Dir(p))
}

func (d *data) addFile(filename, dest string) error {
	// Stat first: opening a FIFO blocks
	fi, err := os.Stat(filename)
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", filename)
	}
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return d.addRegular(dest, fi, fi.Mode(), d.date.clamp(fi.ModTime()), fi.Size(), f)
}

func (d *data) addFileString(contents, dest string) error {
	return d.addRegular(dest, nil, 0644, d.date.now(), int64(len(contents)), strings.NewReader(contents))
}

func (d *data) addRegular(dest string, src fs.FileInfo, mode fs.FileMode, mtime time.Time, size int64, r io.Reader) error {
	p, err := d.reserve(dest, src)
	if err != nil || p == "" {
		return err
	}
	sum := md5.New()
	if err := d.tgz.AddFile(p, mode, mtime, size, io.TeeReader(r, sum)); err != nil {
		return err
	}
	d.setEntry(p, kindFile, src)
	d.size += (uint64(size) + 1023) / 1024
	fmt.Fprintf(&d.md5sums, "%x  %s\n", sum.Sum(nil), p)
	return nil
}

func (d *data) addSymlink(dest string, src fs.FileInfo, target string, mtime time.Time) error {
	p, err := d.reserve(dest, src)
	if err != nil || p == "" {
		return err
	}
	if err := d.tgz.AddSymlink(p, target, mtime); err != nil {
		return err
	}
	d.setEntry(p, kindSymlink, src)
	d.size += (uint64(len(target)) + 1023) / 1024
	return nil
}

// addTree adds the tree rooted at root at the same path, keeping symbolic links.
func (d *data) addTree(root string) error {
	fi, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", root)
	}
	// WalkDir does not descend into a symlinked root unless a separator follows it
	if li, err := os.Lstat(root); err == nil && li.Mode()&fs.ModeSymlink != 0 {
		root += string(filepath.Separator)
	}
	return filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		switch t := e.Type(); {
		case t.IsDir():
			return d.addDirectory(p)
		case t.IsRegular():
			return d.addFile(p, p)
		case t&fs.ModeSymlink != 0:
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			info, err := e.Info()
			if err != nil {
				return err
			}
			return d.addSymlink(p, info, filepath.ToSlash(target), d.date.clamp(info.ModTime()))
		default:
			return fmt.Errorf("%s: unsupported file type %s", p, t)
		}
	})
}

func (d *data) setEntry(p string, k entryKind, src fs.FileInfo) {
	if d.entries == nil {
		d.entries = make(map[string]entryKind)
		d.sources = make(map[string]fs.FileInfo)
	}
	d.entries[p] = k
	if src != nil {
		d.sources[p] = src
	}
}
