// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// arWriter writes the common ar format of deb(5)
type arWriter struct {
	w     io.Writer
	mtime int64
}

func (a *arWriter) add(name string, size int64, r io.Reader) error {
	hdr := fmt.Sprintf("%-16s%-12d%-6d%-6d%-8o%-10d`\n", name, a.mtime, 0, 0, 0100644, size)
	if len(hdr) != 60 {
		return fmt.Errorf("ar member %s: name, size or time does not fit the header", name)
	}
	if _, err := io.WriteString(a.w, hdr); err != nil {
		return err
	}
	if _, err := io.CopyN(a.w, r, size); err != nil {
		return err
	}
	if size%2 == 0 {
		return nil
	}
	_, err := io.WriteString(a.w, "\n")
	return err
}

func (a *arWriter) addFile(name, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	return a.add(name, fi.Size(), f)
}

func (deb *DebPkg) createDebAr(filename string, signature []byte) (err error) {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err == nil {
			return
		}
		// Remove the partial package, but never a device such as /dev/stdout or a link
		if fi, lerr := os.Lstat(filename); lerr == nil && fi.Mode().IsRegular() {
			os.Remove(filename)
		}
	}()

	w := bufio.NewWriter(f)
	ar := arWriter{w: w, mtime: deb.date.now().Unix()}
	if _, err := io.WriteString(w, "!<arch>\n"); err != nil {
		return err
	}
	if err := ar.add("debian-binary", int64(len(debianBinary)), strings.NewReader(debianBinary)); err != nil {
		return fmt.Errorf("add debian-binary: %w", err)
	}
	if err := ar.addFile("control.tar.gz", deb.control.tgz.Name()); err != nil {
		return fmt.Errorf("add control.tar.gz: %w", err)
	}
	if err := ar.addFile("data.tar.gz", deb.data.tgz.Name()); err != nil {
		return fmt.Errorf("add data.tar.gz: %w", err)
	}
	if signature != nil {
		if err := ar.add(digestMember, int64(len(signature)), bytes.NewReader(signature)); err != nil {
			return fmt.Errorf("add %s: %w", digestMember, err)
		}
	}
	return w.Flush()
}
