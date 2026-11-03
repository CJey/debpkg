// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"errors"
)

// ErrClosed is returned when the package is used after Close or Write
var ErrClosed = errors.New("debpkg: closed")

// ErrIO is returned when the intermediate files cannot be created
var ErrIO = errors.New("debpkg: I/O failed")

// check returns the kept error, which takes precedence over ErrClosed.
func (deb *DebPkg) check() error {
	if deb.err != nil {
		return deb.err
	}
	if deb.closed {
		return ErrClosed
	}
	return nil
}

// setError keeps the first error, so that a package is never written with parts missing.
func (deb *DebPkg) setError(err error) error {
	if err != nil && deb.err == nil {
		deb.err = err
	}
	return err
}
