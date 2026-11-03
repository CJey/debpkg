// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/cjey/debpkg/internal/targzip"
)

type control struct {
	tgz       *targzip.TarGzip
	date      sourceDate
	info      controlInfo
	conffiles []string        // absolute paths
	extras    map[string]bool // files added by AddControlExtra*
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
	descr           string         // Long package description, formatted when written
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
	deb.control.info.name = strings.TrimSpace(name)
}

// SetVersion sets the full version, [epoch:]upstream_version[-debian_revision], which takes
// precedence over SetVersionMajor, SetVersionMinor and SetVersionPatch.
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Version
func (deb *DebPkg) SetVersion(version string) {
	deb.control.info.version.full = strings.TrimSpace(version)
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

// SetArchitecture sets the architecture the package installs on (mandatory), such as amd64 or
// arm64 as listed by `dpkg-architecture -L`, or all for an architecture independent package.
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Architecture
func (deb *DebPkg) SetArchitecture(arch string) {
	deb.control.info.architecture = strings.TrimSpace(arch)
}

// SetMaintainer sets the package maintainers name and surname. E.g: "Foo Bar"
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Maintainer
func (deb *DebPkg) SetMaintainer(maintainer string) {
	deb.control.info.maintainer = strings.TrimSpace(maintainer)
}

// SetMaintainerEmail sets the package maintainers email address. E.g: "foo@bar.com"
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Maintainer
func (deb *DebPkg) SetMaintainerEmail(email string) {
	deb.control.info.maintainerEmail = strings.TrimSpace(email)
}

// SetDepends sets the package dependencies. E.g: "lsb-release"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html#s-binarydeps
func (deb *DebPkg) SetDepends(depends string) {
	deb.control.info.depends = strings.TrimSpace(depends)
}

// SetRecommends sets the package recommendations. E.g: "aptitude"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html#s-binarydeps
func (deb *DebPkg) SetRecommends(recommends string) {
	deb.control.info.recommends = strings.TrimSpace(recommends)
}

// SetSuggests sets the package suggestions. E.g: "aptitude"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html#s-binarydeps
func (deb *DebPkg) SetSuggests(suggests string) {
	deb.control.info.suggests = strings.TrimSpace(suggests)
}

// SetConflicts sets one or more conflicting packages. E.g: "nano"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html#s-conflicts
func (deb *DebPkg) SetConflicts(conflicts string) {
	deb.control.info.conflicts = strings.TrimSpace(conflicts)
}

// SetProvides sets the type which the package provides. E.g: "editor"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html#s-virtual
func (deb *DebPkg) SetProvides(provides string) {
	deb.control.info.provides = strings.TrimSpace(provides)
}

// SetReplaces sets the names of packages which will be replaced. E.g: "pico"
// See: https://www.debian.org/doc/debian-policy/ch-relationships.html
func (deb *DebPkg) SetReplaces(replaces string) {
	deb.control.info.replaces = strings.TrimSpace(replaces)
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
	deb.control.info.section = strings.TrimSpace(section)
}

// SetHomepage sets the homepage URL of the package. E.g: "https://github.com/foo/bar"
func (deb *DebPkg) SetHomepage(url string) {
	deb.control.info.homepage = strings.TrimSpace(url)
}

// SetShortDescription sets the single line synopsis (mandatory). E.g: "My awesome foo bar baz tool"
func (deb *DebPkg) SetShortDescription(descr string) {
	deb.control.info.descrShort = strings.TrimSpace(descr)
}

// SetDescription sets the extended description over several lines. It is written in the
// control file format: each line indented by a space, blank lines as " .".
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#s-f-Description
func (deb *DebPkg) SetDescription(descr string) {
	deb.control.info.descr = descr
}

// SetVcsType sets the version control system (Vcs) type for the source package.
// See: https://www.debian.org/doc/manuals/developers-reference/best-pkging-practices.html#s6.2.5.2
func (deb *DebPkg) SetVcsType(vcs VcsType) {
	deb.control.info.vcsType = vcs
}

// SetVcsURL sets the version control system (Vcs) URL for the source package.
// See: https://www.debian.org/doc/manuals/developers-reference/best-pkging-practices.html#s6.2.5.2
func (deb *DebPkg) SetVcsURL(url string) {
	deb.control.info.vcsURL = strings.TrimSpace(url)
}

// SetVcsBrowser sets the version control system (Vcs) browsable source-tree URL for the source package.
// See: https://www.debian.org/doc/manuals/developers-reference/best-pkging-practices.html#s6.2.5.2
func (deb *DebPkg) SetVcsBrowser(url string) {
	deb.control.info.vcsBrowser = strings.TrimSpace(url)
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
	deb.control.info.builtUsing = strings.TrimSpace(info)
}

// SetField sets a field the other Set* functions do not cover, such as Pre-Depends, Breaks or a
// user-defined one like Git-Commit, written after the standard fields and before Description.
// Names are case-insensitive: setting one again replaces its value, an empty value removes it.
// The value is a single line, its leading and trailing whitespace dropped.
// See: https://www.debian.org/doc/debian-policy/ch-controlfields.html#syntax-of-control-files
// and https://www.debian.org/doc/debian-policy/ch-controlfields.html#user-defined-fields
func (deb *DebPkg) SetField(name, value string) error {
	if err := deb.check(); err != nil {
		return err
	}
	value = strings.TrimSpace(value)
	if err := validField(name, value); err != nil {
		return deb.setError(err)
	}
	fields := deb.control.info.fields
	for i, f := range fields {
		if strings.EqualFold(f.name, name) {
			if value == "" {
				deb.control.info.fields = slices.Delete(fields, i, i+1)
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

// statusFields belong to the dpkg database, dpkg refuses them in a package
var statusFields = []string{"Status", "Config-Version", "Conffiles", "Triggers-Pending", "Triggers-Awaited"}

// validField checks the name is neither invalid nor reserved, and the value a single line
func validField(name, value string) error {
	containsName := func(list []string) bool {
		return slices.ContainsFunc(list, func(s string) bool { return strings.EqualFold(name, s) })
	}
	switch {
	case !validFieldName(name):
		return fmt.Errorf("invalid field name %q", name)
	case containsName(ownFields):
		return fmt.Errorf("field %s is set by its own Set* function", name)
	case len(name) > 4 && strings.EqualFold(name[:4], "Vcs-"):
		return fmt.Errorf("field %s is set by SetVcsType, SetVcsURL and SetVcsBrowser", name)
	case containsName(statusFields):
		return fmt.Errorf("field %s is not allowed in a package", name)
	case !singleLine(value):
		return fmt.Errorf("value of field %s is not a single line of text", name)
	}
	return nil
}

// validFieldName reports whether name is printable US-ASCII except space and colon, not
// beginning with # or -
func validFieldName(name string) bool {
	return name != "" && name[0] != '#' && name[0] != '-' &&
		!strings.ContainsFunc(name, func(c rune) bool { return c <= ' ' || c > '~' || c == ':' })
}

// singleLine reports whether s holds no control characters but tabs
func singleLine(s string) bool {
	return !strings.ContainsFunc(s, func(c rune) bool { return c < ' ' && c != '\t' || c == 0x7f })
}

// AddControlExtraString adds a control file, such as a maintainer script, with the content s.
// DOS line endings are converted to UNIX ones, maintainer scripts get mode 0755.
// See: https://www.debian.org/doc/debian-policy/ch-maintainerscripts.html
func (deb *DebPkg) AddControlExtraString(name, s string) error {
	if err := deb.check(); err != nil {
		return err
	}
	return deb.setError(deb.control.addExtra(name, strings.ReplaceAll(s, "\r\n", "\n")))
}

// AddControlExtra is AddControlExtraString with the content read from filename
func (deb *DebPkg) AddControlExtra(name, filename string) error {
	if err := deb.check(); err != nil {
		return err
	}
	b, err := os.ReadFile(filename)
	if err != nil {
		return deb.setError(err)
	}
	return deb.AddControlExtraString(name, string(b))
}

var maintainerScripts = []string{"preinst", "postinst", "prerm", "postrm", "config"}

func (c *control) addExtra(name, content string) error {
	switch {
	case name == "" || name == "." || name == ".." ||
		strings.ContainsFunc(name, func(c rune) bool { return c <= ' ' || c == '/' || c == 0x7f }):
		return fmt.Errorf("invalid control file name %q", name)
	case name == "control" || name == "md5sums":
		return fmt.Errorf("control file %s is generated by debpkg", name)
	case c.extras[name]:
		return fmt.Errorf("control file %s is already added", name)
	}
	mode := fs.FileMode(0644)
	if slices.Contains(maintainerScripts, name) {
		mode = 0755
	}
	if err := c.tgz.AddFile(name, mode, c.date.now(), int64(len(content)), strings.NewReader(content)); err != nil {
		return err
	}
	if c.extras == nil {
		c.extras = make(map[string]bool)
	}
	c.extras[name] = true
	return nil
}

func (c *control) markConfigFile(dest string) error {
	p, err := debPath(dest)
	if err != nil {
		return err
	}
	if p == "" {
		return errors.New("empty config file path")
	}
	p = "/" + p // dpkg requires absolute paths
	// dpkg refuses a conffiles line over 997 bytes, newline included, and trims its trailing
	// whitespace, which would silently make the file no conffile
	switch {
	case len(p) > 996:
		return fmt.Errorf("config file path %s is longer than the 996 bytes dpkg accepts", p)
	case strings.TrimRight(p, " \t\v\f") != p:
		return fmt.Errorf("config file path %q ends with whitespace, which dpkg drops", p)
	}
	if !slices.Contains(c.conffiles, p) {
		c.conffiles = append(c.conffiles, p)
	}
	return nil
}

// verify checks what dpkg rejects, so that an uninstallable package is never written
func (c *control) verify(d *data) error {
	if err := validPackageName(c.info.name); err != nil {
		return err
	}
	if err := validVersion(c.version()); err != nil {
		return err
	}
	if err := validArchitecture(c.info.architecture); err != nil {
		return err
	}
	if c.info.descrShort == "" {
		return errors.New("empty short description")
	}
	// Otherwise "a^ZEssential: yes" would add a field
	if strings.ContainsRune(c.info.descr, 0x1a) {
		return errors.New("extended description contains ^Z, which dpkg reads as a line break")
	}
	for _, f := range c.fields(0) {
		// Vcs-<type> is the only name not checked when set
		if !validFieldName(f.name) {
			return fmt.Errorf("invalid field name %q", f.name)
		}
		if !singleLine(f.value) {
			return fmt.Errorf("value of field %s is not a single line of text", f.name)
		}
	}
	if len(c.conffiles) > 0 && c.extras["conffiles"] {
		return errors.New("conffiles are set both by MarkConfigFile and AddControlExtra")
	}
	for _, p := range c.conffiles {
		switch d.entries[p[1:]] {
		case kindFile, kindSymlink:
		case 0:
			return fmt.Errorf("conffile %s is not in the package", p)
		default:
			return fmt.Errorf("conffile %s is not a regular file", p)
		}
	}
	return nil
}

// validPackageName follows dpkg: an alphanumeric, then alphanumerics and "-+._"
func validPackageName(s string) error {
	if s == "" {
		return errors.New("empty package name")
	}
	for i, c := range s {
		if !isAlnum(c) && (i == 0 || !strings.ContainsRune("-+._", c)) {
			return fmt.Errorf("invalid package name %q", s)
		}
	}
	return nil
}

// validArchitecture follows dpkg: an alphanumeric, then alphanumerics and "-". "source" and
// wildcards such as "any" or "linux-any" match no real system, so dpkg refuses to install them.
func validArchitecture(s string) error {
	if s == "" {
		return errors.New("empty architecture")
	}
	for i, c := range s {
		if !isAlnum(c) && (i == 0 || c != '-') {
			return fmt.Errorf("invalid architecture %q", s)
		}
	}
	if s == "source" || slices.Contains(strings.Split(s, "-"), "any") {
		return fmt.Errorf("architecture %s is not valid for a binary package", s)
	}
	return nil
}

// validVersion follows the checks of dpkg on [epoch:]upstream_version[-debian_revision]
func validVersion(s string) error {
	invalid := func(reason string) error { return fmt.Errorf("invalid version %q: %s", s, reason) }
	v := s
	if i := strings.IndexByte(v, ':'); i >= 0 {
		if _, err := strconv.ParseUint(v[:i], 10, 31); err != nil {
			return invalid("epoch is not a number")
		}
		v = v[i+1:]
	}
	upstream, revision := v, ""
	if i := strings.LastIndexByte(v, '-'); i >= 0 {
		upstream, revision = v[:i], v[i+1:]
		if revision == "" {
			return invalid("empty revision")
		}
	}
	switch {
	case upstream == "":
		return invalid("empty upstream version")
	case upstream[0] < '0' || upstream[0] > '9':
		return invalid("upstream version does not start with a digit")
	case strings.ContainsFunc(upstream, func(c rune) bool { return !isAlnum(c) && !strings.ContainsRune(".+~-:", c) }):
		return invalid("invalid character in upstream version")
	case strings.ContainsFunc(revision, func(c rune) bool { return !isAlnum(c) && !strings.ContainsRune(".+~", c) }):
		return invalid("invalid character in revision")
	}
	return nil
}

func isAlnum(c rune) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
}

// finalize adds the generated control files
func (c *control) finalize(d *data) error {
	add := func(name, content string) error {
		return c.tgz.AddFile(name, 0644, c.date.now(), int64(len(content)), strings.NewReader(content))
	}
	if err := add("control", c.String(d.size)); err != nil {
		return err
	}
	if md5sums := d.md5sums.String(); md5sums != "" {
		if err := add("md5sums", md5sums); err != nil {
			return err
		}
	}
	if len(c.conffiles) > 0 {
		if err := add("conffiles", strings.Join(c.conffiles, "\n")+"\n"); err != nil {
			return err
		}
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

func (c *control) maintainer() string {
	name, email := c.info.maintainer, c.info.maintainerEmail
	switch {
	case email == "":
		return name
	case name == "":
		return "<" + email + ">"
	}
	return name + " <" + email + ">"
}

// fields returns the fields of the control file in order, the empty optional ones dropped
func (c *control) fields(installedSize uint64) []controlField {
	i := &c.info
	vcs := controlField{}
	if i.vcsType != VcsTypeUnset {
		vcs = controlField{"Vcs-" + string(i.vcsType), i.vcsURL}
	}
	fields := []controlField{
		{"Package", i.name},
		{"Version", c.version()},
		{"Architecture", i.architecture},
	}
	for _, f := range []controlField{
		{"Maintainer", c.maintainer()},
		{"Installed-Size", strconv.FormatUint(installedSize, 10)},
		{"Section", i.section},
		{"Priority", string(i.priority)},
		{"Homepage", i.homepage},
		vcs,
		{"Vcs-Browser", i.vcsBrowser},
		{"Built-Using", i.builtUsing},
		{"Depends", i.depends},
		{"Recommends", i.recommends},
		{"Suggests", i.suggests},
		{"Conflicts", i.conflicts},
		{"Provides", i.provides},
		{"Replaces", i.replaces},
	} {
		if f.value != "" {
			fields = append(fields, f)
		}
	}
	fields = append(fields, i.fields...)
	return append(fields, controlField{"Description", i.descrShort})
}

// Create control file for control.tar.gz
func (c *control) String(installedSize uint64) string {
	var b strings.Builder
	for _, f := range c.fields(installedSize) {
		fmt.Fprintf(&b, "%s: %s\n", f.name, f.value)
	}
	writeDescription(&b, c.info.descr)
	return b.String()
}

// writeDescription writes the extended description: each line indented by a space, blank
// lines as " .", and the blank lines around it dropped, since dpkg rejects blank lines.
func writeDescription(b *strings.Builder, descr string) {
	lines := strings.Split(descr, "\n")
	for i, l := range lines {
		// dpkg also takes a line of \v or \f for a blank one
		lines[i] = strings.TrimRight(l, " \t\r\v\f")
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for _, l := range lines {
		if l == "" {
			l = "."
		}
		b.WriteString(" " + l + "\n")
	}
}
