// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"bytes"
	"crypto"
	"crypto/md5"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/clearsign"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

// The digest format of dpkg-sig, which verifies the ar member named after the role.
// See: https://manpages.debian.org/bullseye/dpkg-sig/dpkg-sig.1.en.html
const (
	digestVersion = 4
	digestRole    = "builder"
	digestMember  = "_gpg" + digestRole
)

// digestFile is an ar member as listed in the digest
type digestFile struct {
	name      string
	md5, sha1 []byte
	size      int64
}

func hashFile(name string, r io.Reader) (digestFile, error) {
	m, s := md5.New(), sha1.New()
	n, err := io.Copy(io.MultiWriter(m, s), r)
	if err != nil {
		return digestFile{}, err
	}
	return digestFile{name: name, md5: m.Sum(nil), sha1: s.Sum(nil), size: n}, nil
}

func hashArchive(name, filename string) (digestFile, error) {
	f, err := os.Open(filename)
	if err != nil {
		return digestFile{}, err
	}
	defer f.Close()
	return hashFile(name, f)
}

func digestText(signer string, date time.Time, files []digestFile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Version: %d\nSigner: %s\nDate: %s\nRole: %s\nFiles: \n",
		digestVersion, signer, date.Format(time.ANSIC), digestRole)
	for _, f := range files {
		fmt.Fprintf(&b, "\t%x %x %d %s\n", f.md5, f.sha1, f.size, f.name)
	}
	return b.String()
}

// sign returns the clearsigned digest of the members written before it. The signature is
// randomized by go-crypto and dated now, even when SOURCE_DATE_EPOCH is set.
func (deb *DebPkg) sign(entity *openpgp.Entity) ([]byte, error) {
	now := time.Now()
	key, ok := entity.SigningKey(now)
	if !ok || key.PrivateKey == nil {
		return nil, errors.New("entity has no valid private signing key")
	}
	debianBinaryDigest, err := hashFile("debian-binary", strings.NewReader(debianBinary))
	if err != nil {
		return nil, err
	}
	controlDigest, err := hashArchive("control.tar.gz", deb.control.tgz.Name())
	if err != nil {
		return nil, err
	}
	dataDigest, err := hashArchive("data.tar.gz", deb.data.tgz.Name())
	if err != nil {
		return nil, err
	}
	var signer string
	if id := entity.PrimaryIdentity(); id != nil {
		signer = id.Name
	}

	var buf bytes.Buffer
	w, err := clearsign.Encode(&buf, key.PrivateKey, &packet.Config{DefaultHash: crypto.SHA256})
	if err != nil {
		return nil, err
	}
	text := digestText(signer, now, []digestFile{debianBinaryDigest, controlDigest, dataDigest})
	if _, err := io.WriteString(w, text); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
