// Copyright 2017 Debpkg authors. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
)

// PkgSpecFile represents a single debian package
type PkgSpecFile struct {
	Name            string `yaml:"name"`
	Version         string `yaml:"version"`
	Architecture    string `yaml:"architecture"`
	Maintainer      string `yaml:"maintainer"`
	MaintainerEmail string `yaml:"maintainer_email"`
	Homepage        string `yaml:"homepage"`
	Section         string `yaml:"section"`
	Priority        string `yaml:"priority"`
	Depends         string `yaml:"depends"`
	Recommends      string `yaml:"recommends"`
	Suggests        string `yaml:"suggests"`
	Conflicts       string `yaml:"conflicts"`
	Provides        string `yaml:"provides"`
	Replaces        string `yaml:"replaces"`
	BuiltUsing      string `yaml:"built_using"`
	VcsType         string `yaml:"vcs_type"`
	VcsURL          string `yaml:"vcs_url"`
	VcsBrowser      string `yaml:"vcs_browser"`
	Fields          Fields `yaml:"fields"`
	Description     struct {
		Short string `yaml:"short"`
		Long  string `yaml:"long"`
	} `yaml:"description"`
	Files []struct {
		File       string `yaml:"file"`
		Dest       string `yaml:"dest"`
		Content    string `yaml:"content"`
		ConfigFile bool   `yaml:"conffile"`
	} `yaml:"files"`
	Directories      []string `yaml:"directories"`
	EmptyDirectories []string `yaml:"emptydirs"`
	ControlExtra     struct {
		Preinst  string `yaml:"preinst"`
		Postinst string `yaml:"postinst"`
		Prerm    string `yaml:"prerm"`
		Postrm   string `yaml:"postrm"`
	} `yaml:"control_extra"`
}

// Field is a control field to set with SetField
type Field struct {
	Name  string
	Value string
}

// Fields is decoded from a mapping, keeping its order
type Fields []Field

// UnmarshalYAML implements yaml.Unmarshaler
func (f *Fields) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: fields must be a mapping", node.Line)
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		var value string
		if err := node.Content[i+1].Decode(&value); err != nil {
			return err
		}
		*f = append(*f, Field{Name: node.Content[i].Value, Value: value})
	}
	return nil
}

// Parse decodes a specfile, rejecting unknown keys. Version, section and priority default to
// 0.1.0+dev, misc and optional.
func Parse(data []byte) (*PkgSpecFile, error) {
	cfg := &PkgSpecFile{Version: "0.1.0+dev", Section: "misc", Priority: "optional"}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	return cfg, nil
}
