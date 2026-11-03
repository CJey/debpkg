// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjey/debpkg/internal/test"
)

func TestDigestText(t *testing.T) {
	debianBinaryDigest, err := hashFile("debian-binary", strings.NewReader(debianBinary))
	require.NoError(t, err)
	emptyDigest, err := hashFile("data.tar.gz", strings.NewReader(""))
	require.NoError(t, err)

	digestExpect := "Version: 4\nSigner: Foo Bar <foo@bar.com>\nDate: Tue Nov 14 22:13:20 2023\nRole: builder\nFiles: \n" +
		"\t3cf918272ffa5de195752d73f3da3e5e 7959c969e092f2a5a8604e2287807ac5b1b384ad 4 debian-binary\n" +
		"\td41d8cd98f00b204e9800998ecf8427e da39a3ee5e6b4b0d3255bfef95601890afd80709 0 data.tar.gz\n"
	digest := digestText("Foo Bar <foo@bar.com>", time.Unix(1700000000, 0).UTC(),
		[]digestFile{debianBinaryDigest, emptyDigest})
	assert.Equal(t, digestExpect, digest)
}

// TestWriteSigned verifies the signature is where dpkg-sig looks for it and covers each member
func TestWriteSigned(t *testing.T) {
	entity := test.OpenPGPIdentity(t)
	deb := newTestDeb(t, "debpkg-test-signed")
	deb.SetDescription("very very very very long description")

	// Set version control system info for control file
	deb.SetVcsType(VcsTypeGit)
	deb.SetVcsURL("https://github.com/xor-gate/secdl")
	deb.SetVcsBrowser("https://github.com/xor-gate/secdl")
	deb.SetPriority(PriorityRequired)
	deb.SetConflicts("bash")
	deb.SetProvides("boembats")

	require.NoError(t, deb.AddFile("debpkg.go"))
	f := test.TempFile(t)
	require.NoError(t, deb.WriteSigned(f, entity))
	testReadWithNativeDpkg(t, f)

	d := readDeb(t, f)
	assert.Equal(t, []string{"debian-binary", "control.tar.gz", "data.tar.gz", "_gpgbuilder"}, d.members)
	block, _ := clearsign.Decode(d.raw["_gpgbuilder"])
	require.NotNil(t, block)
	assert.Equal(t, []string{"SHA256"}, block.Headers.Values("Hash"))
	_, err := block.VerifySignature(openpgp.EntityList{entity}, nil)
	require.NoError(t, err)

	digest := string(block.Plaintext)
	assert.Contains(t, digest, "Signer: Debpkg Authors <debpkg-authors@xor-gate.org>\n")
	assert.Contains(t, digest, "Role: builder\n")
	for _, name := range d.members[:3] {
		body := d.raw[name]
		assert.Contains(t, digest, fmt.Sprintf("\t%x %x %d %s\n", md5.Sum(body), sha1.Sum(body), len(body), name))
	}
}

func TestWriteSignedWithoutPrivateKey(t *testing.T) {
	deb := newTestDeb(t, "signed")
	assert.EqualError(t, deb.WriteSigned(test.TempFile(t), nil), "nil signing entity")
	assert.Equal(t, ErrClosed, deb.Close(), "closed like a failed Write")

	var buf bytes.Buffer
	require.NoError(t, test.OpenPGPIdentity(t).Serialize(&buf))
	public, err := openpgp.ReadKeyRing(&buf)
	require.NoError(t, err)
	deb = newTestDeb(t, "signed")
	assert.ErrorContains(t, deb.WriteSigned(test.TempFile(t), public[0]), "no valid private signing key")
}
