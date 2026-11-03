# Changelog

All notable changes to the [debpkg project](https://github.com/cjey/debpkg) will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

* Signing uses `github.com/ProtonMail/go-crypto/openpgp` instead of the deprecated `golang.org/x/crypto/openpgp`; `WriteSigned` takes its `*openpgp.Entity`. The signature is stored in the `_gpgbuilder` member that `dpkg-sig --verify` looks for, hashed with SHA-256, and made with the signing subkey when there is one.
* `GetFilename` returns `<name>_<version>_<architecture>.deb`, the epoch left out.
* `GetArchitecture` maps `ppc64le`, `mipsle`, `mips64le` and `arm` (`armhf`, or `armel` for `GOARM=5`) to their Debian names.
* `SetVar`, `GetVar`, `GetVarWithPrefix` and `ExpandVar` are methods of `DebPkg`. Any variable can be used in the specfile, and `SYSCONFDIR` expands to `/etc`.
* Once a method building the package fails, its error is kept and makes the later calls, `Write` included, fail.
* `Write` refuses an invalid package name, version, architecture or Vcs type, `source` and wildcard architectures such as `any`, an empty short description, a field value that is not a single line, and conffiles that are not in the package. `Write` and `WriteSigned` close the package, also when failing.
* `SetField` refuses the fields of the dpkg database, such as `Status` and `Conffiles`.
* `AddDirectory` keeps symbolic links and refuses other special files. Adding a path twice is an error, unless it is the same file or link from disk again.
* `debpkg` prefixes its errors with its name instead of a timestamp.
* The specfile rejects unknown keys, no longer makes up a name, architecture, maintainer, homepage, `built_using` and description, and supports `vcs_type`, `vcs_url`, `vcs_browser` and `fields`.
* `SOURCE_DATE_EPOCH` sets the timestamps, making unsigned packages reproducible.
* `Installed-Size` is estimated like dpkg-gencontrol does.
* The empty `md5sums` and `conffiles` control files are left out, and so is an empty `Maintainer` field.

### Fixed

* Extended descriptions with blank lines, including lines of mere form feeds, or without a final newline made the package uninstallable.
* A ^Z in the extended description, which dpkg reads as a line break, broke the control file or added fields to it. `Write` refuses it.
* `AddDirectory` added the files of nested directories several times.
* Conffiles were written as relative paths, which dpkg refuses, and a conffile without `dest` in the specfile was not marked.
* A conffile path ending with whitespace, which dpkg drops, silently lost its conffile status, and one over 996 bytes made the package uninstallable. `MarkConfigFile` refuses both.
* An empty destination path produced an entry dpkg refuses.
* A file destination ending with a slash, such as `/opt/foo/`, made the file replace that directory on installation.
* Paths and link targets over 100 bytes or not in ASCII were stored in PAX headers, which dpkg cannot read.
* A failed `Write` removed its output even when not a regular file, such as `/dev/stdout`.
* Errors adding maintainer scripts from the specfile were ignored, and single-line scripts were taken for paths.
* Control files such as `conffiles` were made executable.
* `GetFilename` ignored the version set by `SetVersionMajor`, `SetVersionMinor` and `SetVersionPatch`.
* `WriteSigned` ignored the earlier errors, and panicked like `AddControlExtraString` when `New` had failed.
* The intermediate files were never closed, and were lost when the working directory changed after `New` with a relative temporary directory.
* `debpkg` printed an empty file name without `-o`, and left the intermediate files behind on errors.
* `debpkg` ignored positional arguments, so that `debpkg foo.yml` built `debpkg.yml`.
* `debpkg -o /dev/stdout` wrote its message into the package.

## Initial version - 2017

The first release comes with:

* Setting required fields for the control file
* Adding single files (with optional destination path)
* Adding directories with files recursivley (destination path is not supported)
* Adding empty directories
* Adding control extra files (`preinst`,`postinst`,`prerm`,`postrm`)
* Compression of the data archive with `tar.gz`
* Configuring a custom TMPDIR for intermediate files before `New` calls (defaults to OS)
* Unified configuration file and tool to generate Debian packages
* Marking files as config-files
