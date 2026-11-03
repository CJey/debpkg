// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"

	"github.com/cjey/debpkg/internal/targzip"
)

// DebPkg holds data for a single debian package. Create it with New.
type DebPkg struct {
	control control
	data    data
	vars    map[string]string
	date    sourceDate
	err     error
	closed  bool
}

// New creates a new debian package. The intermediate files are written to tempDir when
// given, which must exist, otherwise to os.TempDir().
func New(tempDir ...string) *DebPkg {
	deb := &DebPkg{vars: defaultVars()}
	date, err := sourceDateFromEnv()
	if err != nil {
		deb.err = err
		return deb
	}
	deb.date, deb.control.date, deb.data.date = date, date, date

	dir := os.TempDir()
	if len(tempDir) > 0 && tempDir[0] != "" {
		dir = tempDir[0]
	}
	// Absolute, so that changing the working directory later keeps the files reachable
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	ctl, err := targzip.NewTempFile(dir)
	if err != nil {
		deb.err = fmt.Errorf("%w: %w", ErrIO, err)
		return deb
	}
	dat, err := targzip.NewTempFile(dir)
	if err != nil {
		ctl.Remove()
		deb.err = fmt.Errorf("%w: %w", ErrIO, err)
		return deb
	}
	deb.control.tgz, deb.data.tgz = ctl, dat
	return deb
}

// Close removes the intermediate files, rendering the package unusable. Write closes the
// package itself; closing it again returns ErrClosed.
func (deb *DebPkg) Close() error {
	if deb.closed {
		return ErrClosed
	}
	deb.closed = true
	var errs []error
	if deb.control.tgz != nil {
		errs = append(errs, deb.control.tgz.Remove())
	}
	if deb.data.tgz != nil {
		errs = append(errs, deb.data.tgz.Remove())
	}
	return errors.Join(errs...)
}

// Write writes the package to filename, or to GetFilename() when empty, and closes it.
func (deb *DebPkg) Write(filename string) error {
	return deb.write(filename, false, nil)
}

// WriteSigned is Write adding a builder signature in the dpkg-sig format, made with the
// signing key of entity.
func (deb *DebPkg) WriteSigned(filename string, entity *openpgp.Entity) error {
	return deb.write(filename, true, entity)
}

func (deb *DebPkg) write(filename string, signed bool, entity *openpgp.Entity) (err error) {
	if err := deb.check(); err != nil {
		return err
	}
	defer func() {
		deb.setError(err)
		deb.Close()
	}()

	if signed && entity == nil {
		return errors.New("nil signing entity")
	}
	if err := deb.control.verify(&deb.data); err != nil {
		return err
	}
	if filename == "" {
		filename = deb.GetFilename()
	}
	if err := deb.control.finalize(&deb.data); err != nil {
		return fmt.Errorf("create control.tar.gz: %w", err)
	}
	if err := deb.control.tgz.Close(); err != nil {
		return fmt.Errorf("close control.tar.gz: %w", err)
	}
	if err := deb.data.tgz.Close(); err != nil {
		return fmt.Errorf("close data.tar.gz: %w", err)
	}
	var signature []byte
	if signed {
		if signature, err = deb.sign(entity); err != nil {
			return fmt.Errorf("sign: %w", err)
		}
	}
	return deb.createDebAr(filename, signature)
}

// GetFilename returns the conventional file name "<name>_<version>_<architecture>.deb",
// the epoch of the version omitted.
func (deb *DebPkg) GetFilename() string {
	version := deb.control.version()
	if i := strings.IndexByte(version, ':'); i >= 0 {
		version = version[i+1:]
	}
	return fmt.Sprintf("%s_%s_%s.deb", deb.control.info.name, version, deb.control.info.architecture)
}

// MarkConfigFile marks dest, a file of the package, as a configuration file
func (deb *DebPkg) MarkConfigFile(dest string) error {
	if err := deb.check(); err != nil {
		return err
	}
	return deb.setError(deb.control.markConfigFile(dest))
}

// AddFile adds the regular file filename to the package at dest, or at filename when dest is
// omitted. Adding the same file again at the same path does nothing.
func (deb *DebPkg) AddFile(filename string, dest ...string) error {
	if err := deb.check(); err != nil {
		return err
	}
	to := filename
	if len(dest) > 0 && dest[0] != "" {
		to = dest[0]
	}
	return deb.setError(deb.data.addFile(filename, to))
}

// AddFileString adds a file holding contents at dest, with mode 0644
func (deb *DebPkg) AddFileString(contents, dest string) error {
	if err := deb.check(); err != nil {
		return err
	}
	return deb.setError(deb.data.addFileString(contents, dest))
}

// AddEmptyDirectory adds an empty directory to the package
func (deb *DebPkg) AddEmptyDirectory(dir string) error {
	if err := deb.check(); err != nil {
		return err
	}
	return deb.setError(deb.data.addDirectory(dir))
}

// AddDirectory adds the tree rooted at dir to the package at the same path, files keeping their
// mode, directories with mode 0755 and symbolic links as links. A file or link already added from
// the same source is not added again.
func (deb *DebPkg) AddDirectory(dir string) error {
	if err := deb.check(); err != nil {
		return err
	}
	return deb.setError(deb.data.addTree(dir))
}

// sourceDate is the time given by SOURCE_DATE_EPOCH, zero when unset.
// See: https://reproducible-builds.org/specs/source-date-epoch/
type sourceDate time.Time

func sourceDateFromEnv() (sourceDate, error) {
	v := os.Getenv("SOURCE_DATE_EPOCH")
	if v == "" {
		return sourceDate{}, nil
	}
	sec, err := strconv.ParseInt(v, 10, 64)
	if err != nil || sec < 0 {
		return sourceDate{}, fmt.Errorf("debpkg: invalid SOURCE_DATE_EPOCH %q", v)
	}
	return sourceDate(time.Unix(sec, 0).UTC()), nil
}

// now returns the timestamp of generated entries.
func (s sourceDate) now() time.Time {
	if time.Time(s).IsZero() {
		return time.Now()
	}
	return time.Time(s)
}

// clamp caps the modification time of files taken from disk.
func (s sourceDate) clamp(t time.Time) time.Time {
	if !time.Time(s).IsZero() && t.After(time.Time(s)) {
		return time.Time(s)
	}
	return t
}
