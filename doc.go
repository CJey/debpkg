// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// Package debpkg implements creation of (gpg signed) debian packages
//
// # Overview
//
// The most basic debian package is created as follows (without error checking):
//
//	deb := debpkg.New()
//	defer deb.Close()
//
//	deb.SetName("foobar")
//	deb.SetVersion("1.2.3")
//	deb.SetArchitecture("amd64")
//	deb.SetMaintainer("Foo Bar")
//	deb.SetMaintainerEmail("foo@bar.com")
//	deb.SetHomepage("http://foobar.com")
//
//	deb.SetShortDescription("Minimal foo bar package")
//	deb.SetDescription("Foo bar package doesn't do anything")
//
//	deb.AddFile("build/foobar", "/usr/bin/foobar")
//
//	deb.Write("foobar.deb")
//
// # Errors
//
// Once a method building the package fails, its error is kept: the later calls, Write
// included, fail with it. A package is thus never written with parts silently missing.
//
// # Reproducible builds
//
// When SOURCE_DATE_EPOCH is set, it is the time of the generated entries and caps the
// modification time of the files taken from disk, so that the same input gives the same
// package. Signatures are randomized and remain different.
package debpkg
