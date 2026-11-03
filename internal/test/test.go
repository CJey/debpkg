// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

// TempFile returns the path of a package file in a temporary directory of the test
func TempFile(t *testing.T) string {
	return filepath.Join(t.TempDir(), "test.deb")
}

// WriteTempFile writes data to name in a temporary directory of the test and returns its path
func WriteTempFile(t *testing.T, name, data string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

var identity = sync.OnceValues(func() (*openpgp.Entity, error) {
	// Created in the past, so that a signature made right away is never dated before it
	cfg := &packet.Config{Algorithm: packet.PubKeyAlgoEdDSA, Time: func() time.Time { return time.Unix(1e9, 0) }}
	return openpgp.NewEntity("Debpkg Authors", "", "debpkg-authors@xor-gate.org", cfg)
})

// OpenPGPIdentity returns a signing identity shared by the tests
func OpenPGPIdentity(t *testing.T) *openpgp.Entity {
	t.Helper()
	e, err := identity()
	if err != nil {
		t.Fatal(err)
	}
	return e
}
