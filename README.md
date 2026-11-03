# debpkg

[![Go Reference](https://pkg.go.dev/badge/github.com/cjey/debpkg.svg)](https://pkg.go.dev/github.com/cjey/debpkg)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

debpkg is a pure [Go](https://go.dev) library to create [Debian](https://debian.org) Linux packages. The package can be directly imported in your Go project, or the `debpkg` cli tool can be run with a custom debpkg YAML specfile (just like a single [RPM spec file](http://ftp.rpm.org/max-rpm/s1-rpm-build-creating-spec-file.html)). It has no dependencies outside Go modules and doesn't use exec wrappers. Therefore it is able to generate packages from non Linux systems like macOS, *BSD and Windows without the Perl dependency (as with the official [debbuild](https://github.com/debbuild/debbuild)). The Debian package build system is universal and flexible, which can be a pro or a con: packaging just a small bunch of files with a simple configuration requires to understand a fairly complex beast, and to dig deep into the Debian package management documentation. debpkg is not able to resolve all the advanced packaging requirements, for which one must use the official `debbuild` and `debhelper`. It is created for the simple case of converting a directory structure to be installable and maintained the standard Debian way.

This project is a fork of [xor-gate/debpkg](https://github.com/xor-gate/debpkg).

## Features

- Create debian packages from files, inline content and directory trees (symbolic links are kept)
- Create packages from a `debpkg.yml` specfile
- Add custom control files (preinst, postinst, prerm, postrm etcetera)
- Mark configuration files (conffiles)
- Set further control fields (`Pre-Depends`, `Breaks`, user-defined ones etcetera) with `SetField`
- Sign packages in the format of [dpkg-sig](https://manpages.debian.org/bullseye/dpkg-sig/dpkg-sig.1.en.html)
- Build reproducible packages with [`SOURCE_DATE_EPOCH`](https://reproducible-builds.org/specs/source-date-epoch/)

`Write` refuses what dpkg would refuse (an invalid package name, version or architecture, a missing short description, conffiles that are not in the package), so that an uninstallable package is never written. Once a method building the package fails, its error is kept and makes `Write` fail, so that a package is never written with parts silently missing.

It is currently not possible to use `debpkg` as a framework to manipulate and introspect individual Debian package objects ([see issue #26](https://github.com/xor-gate/debpkg/issues/26)), as it is only capable of creating packages.

## Why this package was created

This package was created due to the lack of a way to debianize from other platforms (Windows, macOS, *BSD), because the whole Debian package management system is glued together with Perl scripts and uses a bunch of Perl modules, which isn't easy to use in a CI/CD pipeline for Go projects.

Converting a simple directory structure with files into a Debian package is difficult without knowing the `deb` file internals.

This package is heavily inspired by [godeb](https://github.com/niemeyer/godeb) and [CPackDeb](https://cmake.org/cmake/help/v3.5/module/CPackDeb.html). It is liberally [licensed](LICENSE).

## Installation

`go install github.com/cjey/debpkg/cmd/debpkg@latest`

## Usage

```
debpkg [-c debpkg.yml] [-o output.deb] [-v version]
```

The output file defaults to `<name>_<version>_<architecture>.deb`. The version given by `-v`, or else by the `DEBPKG_VERSION` environment variable, overrides the one of the specfile.

## debpkg.yml specfile

The specfile is written in the [YAML markup format](https://yaml.org/). It controls the package information and data. Unknown keys are rejected.

A simple example is listed below:

```yaml
# debpkg.yml specfile
name: foobar
version: 7.6.5
architecture: all
maintainer: Foo Bar
maintainer_email: foo@bar.com
homepage: https://github.com/cjey/debpkg
description:
  short: This package is just a test
  long: |
    This package tests the working of debpkg.
    And can wrap multiple
    lines.

    And multiple paragraphs.
```

`name`, `architecture` and `description.short` are required; `version`, `section` and `priority` default to `0.1.0+dev`, `misc` and `optional`. The other keys are:

```yaml
depends: lsb-release              # also recommends, suggests, conflicts, provides and replaces
built_using: gcc-4.6 (= 4.6.0-11)
vcs_type: Git
vcs_url: https://github.com/cjey/debpkg.git
vcs_browser: https://github.com/cjey/debpkg
fields:                           # further control fields, written in this order
  Pre-Depends: dpkg (>= 1.19)
files:
  - file: build/foobar            # a file on disk, placed at its own path unless dest is given
    dest: "{{.BINDIR}}/foobar"
  - content: |                    # inline content, dest is required
      key = value
    dest: "{{.SYSCONFDIR}}/foobar.conf"
    conffile: true                # mark as a configuration file
directories:                      # added recursively at the same path
  - usr/share/foobar
emptydirs:
  - /var/cache/foobar
control_extra:                    # preinst, postinst, prerm and postrm
  postinst: |                     # inline when starting with #! or spanning several lines
    #!/bin/sh
    echo installed
  prerm: scripts/prerm            # otherwise read from this file
```

Relative paths are taken from the working directory. Files on disk keep their mode, inline content gets mode 0644 and directories 0755. A file of `directories` can be listed in `files` too, so as to mark it as a conffile.

Before being parsed, the specfile is expanded as a Go template with the installation variables: `{{.INSTALLPREFIX}}` (`/usr`), `{{.BINDIR}}` (`/usr/bin`), `{{.SBINDIR}}` (`/usr/sbin`), `{{.SYSCONFDIR}}` (`/etc`), `{{.DATAROOTDIR}}` (`/usr/share`), and any variable set with `SetVar`.

## Mentions

This project originates from an in-company implementation sponsored by [@dualinventive](https://github.com/dualinventive) in 2016-2017, with help from colleague [@rikvdh](https://github.com/rikvdh).

I would like to mention some other noticeable projects for debian package management:

* https://github.com/Debian/dh-make-golang
* https://github.com/niemeyer/godeb
* https://github.com/smira/aptly
* https://github.com/esell/deb-simple
* https://github.com/paultag/go-debian
* https://github.com/jordansissel/fpm
* https://github.com/laher/debgo-v0.2
* https://github.com/debber/debber-v0.3
* https://github.com/laher/goxc
* https://github.com/knqyf263/go-deb-version

Other packaging tools in golang:

* https://github.com/google/rpmpack

## Debugging

When debugging on a Debian system the following commands are useful:

* Print package info: `dpkg --info <debfile>`
* Parse the control file strictly: `dpkg-deb --show <debfile>`
* Extract data.tar.gz: `dpkg --extract <debfile> data`
* Verbose extract data.tar.gz: `dpkg --vextract <debfile> data`
* Extract control.tar.gz: `dpkg --control <debfile> control`
* Show .deb file contents: `dpkg-deb --contents <debfile>`
* Install .deb file manually without repository: `dpkg -i <debfile>`

## Further documentation

More information can be gathered from the Debian and Ubuntu documentation:

* [dpkg manpage](https://manpages.debian.org/dpkg)
* [dpkg-deb manpage](https://manpages.debian.org/dpkg-deb)
* [dpkg-sig manpage](https://manpages.debian.org/bullseye/dpkg-sig/dpkg-sig.1.en.html)
* [Debian New Maintainers' Guide](https://www.debian.org/doc/manuals/maint-guide/)
* [Debian Policy Manual](https://www.debian.org/doc/debian-policy/)
* [Setting up signed APT repository with Reprepro](https://wiki.debian.org/SettingUpSignedAptRepositoryWithReprepro)
* [Create authenticated repository](https://help.ubuntu.com/community/CreateAuthenticatedRepository)

## License

[MIT](LICENSE)
