// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package debpkg

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cjey/debpkg/internal/config"
)

// Config loads settings from a debpkg.yml specfile, its variables expanded first. It also resets
// the standard fields the specfile leaves out, so Set* calls meant to override it come after it.
func (deb *DebPkg) Config(filename string) error {
	if err := deb.check(); err != nil {
		return err
	}
	return deb.setError(deb.config(filename))
}

func (deb *DebPkg) config(filename string) error {
	b, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	expanded, err := deb.ExpandVar(string(b))
	if err != nil {
		return fmt.Errorf("expand config file: %w", err)
	}
	cfg, err := config.Parse([]byte(expanded))
	if err != nil {
		return err
	}

	deb.SetSection(cfg.Section)
	deb.SetPriority(Priority(cfg.Priority))
	deb.SetName(cfg.Name)
	deb.SetVersion(cfg.Version)
	deb.SetArchitecture(cfg.Architecture)
	deb.SetMaintainer(cfg.Maintainer)
	deb.SetMaintainerEmail(cfg.MaintainerEmail)
	deb.SetHomepage(cfg.Homepage)
	deb.SetShortDescription(cfg.Description.Short)
	deb.SetDescription(cfg.Description.Long)
	deb.SetBuiltUsing(cfg.BuiltUsing)
	deb.SetDepends(cfg.Depends)
	deb.SetRecommends(cfg.Recommends)
	deb.SetSuggests(cfg.Suggests)
	deb.SetConflicts(cfg.Conflicts)
	deb.SetProvides(cfg.Provides)
	deb.SetReplaces(cfg.Replaces)
	deb.SetVcsType(VcsType(cfg.VcsType))
	deb.SetVcsURL(cfg.VcsURL)
	deb.SetVcsBrowser(cfg.VcsBrowser)
	for _, f := range cfg.Fields {
		if err := deb.SetField(f.Name, f.Value); err != nil {
			return err
		}
	}

	for _, file := range cfg.Files {
		dest := file.Dest
		switch {
		case file.File != "" && file.Content != "":
			return fmt.Errorf("file %s has both 'file' and 'content'", file.File)
		case file.File != "":
			if err := deb.AddFile(file.File, dest); err != nil {
				return fmt.Errorf("add file %s: %w", file.File, err)
			}
			if dest == "" {
				dest = file.File
			}
		case file.Content != "":
			if err := deb.AddFileString(file.Content, dest); err != nil {
				return fmt.Errorf("add file %s: %w", dest, err)
			}
		default:
			return errors.New("a file needs either 'file' or 'content'")
		}
		if file.ConfigFile {
			if err := deb.MarkConfigFile(dest); err != nil {
				return fmt.Errorf("mark config file %s: %w", dest, err)
			}
		}
	}

	for _, dir := range cfg.Directories {
		if err := deb.AddDirectory(dir); err != nil {
			return fmt.Errorf("add directory %s: %w", dir, err)
		}
	}
	for _, dir := range cfg.EmptyDirectories {
		if err := deb.AddEmptyDirectory(dir); err != nil {
			return fmt.Errorf("add directory %s: %w", dir, err)
		}
	}

	for _, script := range []struct{ name, value string }{
		{"preinst", cfg.ControlExtra.Preinst},
		{"postinst", cfg.ControlExtra.Postinst},
		{"prerm", cfg.ControlExtra.Prerm},
		{"postrm", cfg.ControlExtra.Postrm},
	} {
		if script.value == "" {
			continue
		}
		// A script is given inline when it has a shebang or several lines, otherwise by its path
		if strings.HasPrefix(script.value, "#!") || strings.Contains(script.value, "\n") {
			err = deb.AddControlExtraString(script.name, script.value)
		} else {
			err = deb.AddControlExtra(script.name, script.value)
		}
		if err != nil {
			return fmt.Errorf("add %s: %w", script.name, err)
		}
	}
	return nil
}
