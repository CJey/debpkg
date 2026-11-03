// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package targzip

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"
)

// TarGzip writes a .tar.gz temporary file whose entries are owned by root.
type TarGzip struct {
	f      *os.File
	bw     *bufio.Writer // gzip flushes small blocks, batch them into fewer writes
	gw     *gzip.Writer
	tw     *tar.Writer
	closed bool
}

// NewTempFile creates the archive as a new temporary file in dir.
func NewTempFile(dir string) (*TarGzip, error) {
	f, err := os.CreateTemp(dir, "debpkg")
	if err != nil {
		return nil, err
	}
	bw := bufio.NewWriterSize(f, 64<<10)
	gw := gzip.NewWriter(bw)
	return &TarGzip{f: f, bw: bw, gw: gw, tw: tar.NewWriter(gw)}, nil
}

// AddFile adds a regular file with the size bytes read from r.
func (t *TarGzip) AddFile(name string, mode fs.FileMode, mtime time.Time, size int64, r io.Reader) error {
	hdr := &tar.Header{Typeflag: tar.TypeReg, Name: name, Mode: tarMode(mode), Size: size, ModTime: mtime}
	if err := t.writeHeader(hdr); err != nil {
		return err
	}
	if _, err := io.CopyN(t.tw, r, size); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

// AddDirectory adds a directory with mode 0755.
func (t *TarGzip) AddDirectory(name string, mtime time.Time) error {
	return t.writeHeader(&tar.Header{Typeflag: tar.TypeDir, Name: name, Mode: 0755, ModTime: mtime})
}

// AddSymlink adds a symbolic link to target.
func (t *TarGzip) AddSymlink(name, target string, mtime time.Time) error {
	return t.writeHeader(&tar.Header{Typeflag: tar.TypeSymlink, Name: name, Linkname: target, Mode: 0777, ModTime: mtime})
}

func (t *TarGzip) writeHeader(hdr *tar.Header) error {
	// dpkg reads GNU long names, but not the PAX headers Go picks for long or non-ASCII names
	hdr.Format = tar.FormatGNU
	hdr.Uname, hdr.Gname = "root", "root"
	return t.tw.WriteHeader(hdr)
}

// Close completes the archive and closes the file. Later calls do nothing.
func (t *TarGzip) Close() error {
	if t.closed {
		return nil
	}
	t.closed = true
	return errors.Join(t.tw.Close(), t.gw.Close(), t.bw.Flush(), t.f.Close())
}

// Name returns the path of the temporary file.
func (t *TarGzip) Name() string {
	return t.f.Name()
}

// Remove closes the file, without completing the archive if still open, and deletes it.
func (t *TarGzip) Remove() error {
	if !t.closed {
		t.closed = true
		t.f.Close()
	}
	return os.Remove(t.f.Name())
}

// tarMode keeps the permission, setuid, setgid and sticky bits of m.
func tarMode(m fs.FileMode) int64 {
	mode := int64(m.Perm())
	if m&fs.ModeSetuid != 0 {
		mode |= 04000
	}
	if m&fs.ModeSetgid != 0 {
		mode |= 02000
	}
	if m&fs.ModeSticky != 0 {
		mode |= 01000
	}
	return mode
}
