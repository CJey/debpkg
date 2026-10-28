// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"fmt"
	"os"

	"math"
	"strings"

	"github.com/cjey/debpkg/internal/targzip"
)

type control struct {
	tgz                *targzip.TarGzip
	info               controlInfo
	conffiles          string // List of configuration-files
	hasCustomConffiles bool
}

type controlInfoVersion struct {
	full  string // Full version string. E.g "0.1.2"
	major uint   // Major version number
	minor uint   // Minor version number
	patch uint   // Patch version number
}

type controlInfo struct {
	name            string
	version         controlInfoVersion
	architecture    string
	maintainer      string
	maintainerEmail string
	homepage        string
	depends         string
	recommends      string
	suggests        string
	conflicts       string
	provides        string
	replaces        string
	section         string
	priority        Priority
	descrShort      string         // Short package description
	descr           string         // Long package description
	vcsType         VcsType        // E.g: "Svn", "Git" etcetera
	vcsURL          string         // E.g: git@github.com:cjey/debpkg.git
	vcsBrowser      string         // E.g: https://github.com/cjey/debpkg
	builtUsing      string         // E.g: gcc-4.6 (= 4.6.0-11)
	fields          []controlField // Set by SetField, in the order set
}

type controlField struct {
	name  string
	value string
}

// SetName sets the name of the binary package (mandatory)
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Package
func (deb *DebPkg) SetName(name string) {
	deb.control.info.name = name
}

// SetVersion sets the full version string (mandatory), or use SetVersion* functions for "major.minor.patch"
// The upstream_version may contain only alphanumerics ( A-Za-z0-9 ) and the characters . + - : ~
//
//	(full stop, plus, hyphen, colon, tilde) and should start with a digit.
//
// NOTE: When the full string is set the other SetVersion* function calls are ignored
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version
func (deb *DebPkg) SetVersion(version string) {
	deb.control.info.version.full = version
}

// SetVersionMajor sets the version major number
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version
func (deb *DebPkg) SetVersionMajor(major uint) {
	deb.control.info.version.major = major
}

// SetVersionMinor sets the version minor number
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version
func (deb *DebPkg) SetVersionMinor(minor uint) {
	deb.control.info.version.minor = minor
}

// SetVersionPatch sets the version patch level
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version
func (deb *DebPkg) SetVersionPatch(patch uint) {
	deb.control.info.version.patch = patch
}

// SetArchitecture sets the architecture of the package where it can be installed.
//
//	E.g "i386, amd64, arm, any, all". See `dpkg-architecture -L` for all supported.
//
// Architecture: any
//
//	The generated binary package is an architecture dependent one usually in a compiled language.
//
// Architecture: all
//
//	The generated binary package is an architecture independent one usually consisting of text,
//	images, or scripts in an interpreted language.
//
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Architecture
// And: http://man7.org/linux/man-pages/man1/dpkg-architecture.1.html
func (deb *DebPkg) SetArchitecture(arch string) {
	deb.control.info.architecture = arch
}

// SetMaintainer (mandatory), sets the package maintainers name and surname. E.g: "Foo Bar"
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Maintainer
func (deb *DebPkg) SetMaintainer(maintainer string) {
	deb.control.info.maintainer = maintainer
}

// SetMaintainerEmail sets the package maintainers email address. E.g: "foo@bar.com"
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Maintainer
func (deb *DebPkg) SetMaintainerEmail(email string) {
	deb.control.info.maintainerEmail = email
}

// SetDepends sets the package dependencies. E.g: "lsb-release"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html#s-binarydeps
func (deb *DebPkg) SetDepends(depends string) {
	deb.control.info.depends = depends
}

// SetRecommends sets the package recommendations. E.g: "aptitude"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html#s-binarydeps
func (deb *DebPkg) SetRecommends(recommends string) {
	deb.control.info.recommends = recommends
}

// SetSuggests sets the package suggestions. E.g: "aptitude"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html#s-binarydeps
func (deb *DebPkg) SetSuggests(suggests string) {
	deb.control.info.suggests = suggests
}

// SetConflicts sets one or more conflicting packages. E.g: "nano"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html#s-conflicts
func (deb *DebPkg) SetConflicts(conflicts string) {
	deb.control.info.conflicts = conflicts
}

// SetProvides sets the type which the package provides. E.g: "editor"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html#s-virtual
func (deb *DebPkg) SetProvides(provides string) {
	deb.control.info.provides = provides
}

// SetReplaces sets the names of packages which will be replaced. E.g: "pico"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html
func (deb *DebPkg) SetReplaces(replaces string) {
	deb.control.info.replaces = replaces
}

// SetPriority (recommended). Default set to debpkg.PriorityUnset
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Priority
// And: https://www.debian.org/doc/debian-policy/ch-archive.html#s-priorities
func (deb *DebPkg) SetPriority(priority Priority) {
	deb.control.info.priority = priority
}

// SetSection (recommended). E.g: editors
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Section
// And: https://www.debian.org/doc/debian-policy/ch-archive.html#s-subsections
func (deb *DebPkg) SetSection(section string) {
	deb.control.info.section = section
}

// SetHomepage sets the homepage URL of the package. E.g: "https://github.com/foo/bar"
func (deb *DebPkg) SetHomepage(url string) {
	deb.control.info.homepage = url
}

// SetShortDescription sets the single line synopsis. E.g: "My awesome foo bar baz tool"
func (deb *DebPkg) SetShortDescription(descr string) {
	deb.control.info.descrShort = descr
}

// SetDescription sets the extended description over several lines. E.g:
// "This tool will calculation the most efficient way to world domination"
// NOTE: The debian control file has a special formatting of the long description
//
//	this function replaces newlines with a newline and a space.
func (deb *DebPkg) SetDescription(descr string) {
	deb.control.info.descr = " " + strings.Replace(descr, "\n", "\n ", -1)
}

// SetVcsType sets the version control system (Vcs) type for the source package.
// See: https://www.debian.org/doc/manuals/developers-reference/best-pkging-practices.html#s6.2.5.2
func (deb *DebPkg) SetVcsType(vcs VcsType) {
	deb.control.info.vcsType = vcs
}

// SetVcsURL sets the version control system (Vcs) URL for the source package.
// See: https://www.debian.org/doc/manuals/developers-reference/best-pkging-practices.html#s6.2.5.2
func (deb *DebPkg) SetVcsURL(url string) {
	deb.control.info.vcsURL = url
}

// SetVcsBrowser sets the version control system (Vcs) browsable source-tree URL for the source package.
// See: https://www.debian.org/doc/manuals/developers-reference/best-pkging-practices.html#s6.2.5.2
func (deb *DebPkg) SetVcsBrowser(url string) {
	deb.control.info.vcsBrowser = url
}

// SetBuiltUsing incorporate parts of other packages when built but do not have to depend on those packages.
// A package using the source code from the gcc-4.6-source binary package built from the gcc-4.6 source package
// would have this field in its control file:
//
//	Built-Using: gcc-4.6 (= 4.6.0-11)
//
// A package including binaries from grub2 and loadlin would have this field in its control file:
//
//	Built-Using: grub2 (= 1.99-9), loadlin (= 1.6e-1)
//
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html#s-built-using
func (deb *DebPkg) SetBuiltUsing(info string) {
	deb.control.info.builtUsing = info
}

// SetField sets a field the other Set* functions do not cover, such as Pre-Depends, Breaks or a
// user-defined one like Git-Commit, written after the standard fields and before Description.
// Names are case-insensitive: setting one again replaces its value, an empty value removes it.
// The value is a single line, its leading and trailing whitespace dropped.
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#syntax-of-control-files
// and https://www.debian.org/doc/debian-policy/ch-controlfields.html#user-defined-fields
func (deb *DebPkg) SetField(name, value string) error {
	value = strings.TrimSpace(value)
	if err := validField(name, value); err != nil {
		return err
	}
	var fields = deb.control.info.fields
	for i, f := range fields {
		if strings.EqualFold(f.name, name) {
			if value == "" {
				deb.control.info.fields = append(fields[:i], fields[i+1:]...)
			} else {
				fields[i].value = value
			}
			return nil
		}
	}
	if value != "" {
		deb.control.info.fields = append(fields, controlField{name: name, value: value})
	}
	return nil
}

// ownFields the fields written by the other Set* functions, besides Vcs-*
var ownFields = []string{"Package", "Version", "Architecture", "Maintainer", "Installed-Size", "Section",
	"Priority", "Homepage", "Built-Using", "Depends", "Recommends", "Suggests", "Conflicts", "Provides",
	"Replaces", "Description"}

// validField a name is printable US-ASCII except space and colon, not beginning with # or -,
// and not one of the fields written by the other Set* functions
func validField(name, value string) error {
	if name == "" || name[0] == '#' || name[0] == '-' {
		return fmt.Errorf("invalid field name %q", name)
	}
	for _, c := range name {
		if c <= ' ' || c > '~' || c == ':' {
			return fmt.Errorf("invalid field name %q", name)
		}
	}
	for _, own := range ownFields {
		if strings.EqualFold(name, own) {
			return fmt.Errorf("field %s is set by its own Set* function", name)
		}
	}
	if len(name) > 4 && strings.EqualFold(name[:4], "Vcs-") {
		return fmt.Errorf("field %s is set by SetVcsType, SetVcsURL and SetVcsBrowser", name)
	}
	if strings.ContainsFunc(value, func(c rune) bool { return c < ' ' && c != '\t' || c == 0x7f }) {
		return fmt.Errorf("value of field %s is not a single line of text", name)
	}
	return nil
}

// AddControlExtraString is the same as AddControlExtra except it uses a string input.
// the files have possible DOS line-endings replaced by UNIX line-endings
func (deb *DebPkg) AddControlExtraString(name, s string) error {
	if name == "conffiles" {
		deb.control.hasCustomConffiles = true
	}
	s = strings.Replace(s, "\r\n", "\n", -1)
	return deb.control.tgz.AddFileFromBuffer(name, []byte(s), 0755)
}

// AddControlExtra allows the advanced user to add custom script to the control.tar.gz Typical usage is
//
//	for preinst, postinst, postrm, prerm: https://www.debian.org/doc/debian-policy/ch-maintainerscripts.html
//
// And: https://www.debian.org/doc/manuals/maint-guide/dother.en.html#maintscripts
// the files have possible DOS line-endings replaced by UNIX line-endings
func (deb *DebPkg) AddControlExtra(name, filename string) error {
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return deb.AddControlExtraString(name, string(b))
}

// verify the control file for validity
func (c *control) verify() error {
	if c.info.name == "" {
		return fmt.Errorf("empty package name")
	}
	if c.info.architecture == "" {
		return fmt.Errorf("empty architecture")
	}
	return nil
}

func (c *control) markConfigFile(dest string) error {
	if dest == "" {
		return fmt.Errorf("config file cannot be empty")
	}
	c.conffiles += dest + "\n"
	return nil
}

// finalizeControlFile creates the actual control-file, adds MD5-sums and stores
// config-files
func (c *control) finalizeControlFile(d *data) error {
	if !c.hasCustomConffiles {
		if err := c.tgz.AddFileFromBuffer("conffiles", []byte(c.conffiles), 0); err != nil {
			return err
		}
	}
	controlFile := []byte(c.String(d.tgz.Written()))
	if err := c.tgz.AddFileFromBuffer("control", controlFile, 0); err != nil {
		return err
	}
	if err := c.tgz.AddFileFromBuffer("md5sums", []byte(d.md5sums), 0); err != nil {
		return err
	}
	return nil
}

// Generate version string (e.g "1.2.3") from major,minor patch or from full version
func (c *control) version() string {
	if c.info.version.full != "" {
		return c.info.version.full
	}
	return fmt.Sprintf("%d.%d.%d",
		c.info.version.major,
		c.info.version.minor,
		c.info.version.patch)
}

func (c *control) size() int64 {
	return c.tgz.Size()
}

// Create control file for control.tar.gz
func (c *control) String(installedSize uint64) string {
	var o strings.Builder

	o.WriteString(fmt.Sprintf("Package: %s\n", c.info.name))
	o.WriteString(fmt.Sprintf("Version: %s\n", c.version()))
	o.WriteString(fmt.Sprintf("Architecture: %s\n", c.info.architecture))
	o.WriteString(fmt.Sprintf("Maintainer: %s <%s>\n",
		c.info.maintainer,
		c.info.maintainerEmail))
	o.WriteString(fmt.Sprintf("Installed-Size: %d\n", uint64(math.Ceil(float64(installedSize)/1024))))

	if c.info.section != "" {
		o.WriteString(fmt.Sprintf("Section: %s\n", c.info.section))
	}
	if c.info.priority != PriorityUnset {
		o.WriteString(fmt.Sprintf("Priority: %s\n", c.info.priority))
	}
	if c.info.homepage != "" {
		o.WriteString(fmt.Sprintf("Homepage: %s\n", c.info.homepage))
	}
	if c.info.vcsType != VcsTypeUnset && c.info.vcsURL != "" {
		o.WriteString(fmt.Sprintf("Vcs-%s: %s\n", c.info.vcsType, c.info.vcsURL))
	}
	if c.info.vcsBrowser != "" {
		o.WriteString(fmt.Sprintf("Vcs-Browser: %s\n", c.info.vcsBrowser))
	}
	if c.info.builtUsing != "" {
		o.WriteString(fmt.Sprintf("Built-Using: %s\n", c.info.builtUsing))
	}

	if c.info.depends != "" {
		o.WriteString(fmt.Sprintf("Depends: %s\n", c.info.depends))
	}
	if c.info.recommends != "" {
		o.WriteString(fmt.Sprintf("Recommends: %s\n", c.info.recommends))
	}
	if c.info.suggests != "" {
		o.WriteString(fmt.Sprintf("Suggests: %s\n", c.info.suggests))
	}
	if c.info.conflicts != "" {
		o.WriteString(fmt.Sprintf("Conflicts: %s\n", c.info.conflicts))
	}
	if c.info.provides != "" {
		o.WriteString(fmt.Sprintf("Provides: %s\n", c.info.provides))
	}
	if c.info.replaces != "" {
		o.WriteString(fmt.Sprintf("Replaces: %s\n", c.info.replaces))
	}
	for _, f := range c.info.fields {
		o.WriteString(fmt.Sprintf("%s: %s\n", f.name, f.value))
	}

	o.WriteString(fmt.Sprintf("Description: %s\n", c.info.descrShort))
	o.WriteString(fmt.Sprintf("%s", c.info.descr))

	return o.String()
}
