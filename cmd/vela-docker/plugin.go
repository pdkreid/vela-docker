// Copyright (c) 2020 Target Brands, Inc. All rights reserved.
//
// Use of this source code is governed by the LICENSE file in this repository.

package main

import (
	"fmt"
	"os/exec"

	"github.com/spf13/afero"

	"github.com/sirupsen/logrus"
)

var appFS = afero.NewOsFs()

// Plugin represents the configuration loaded for the plugin.
type Plugin struct {
	// build arguments loaded for the plugin
	Build *Build
	// image arguments loaded for the plugin
	Image *Image
	// registry arguments loaded for the plugin
	Registry *Registry
	// repo arguments loaded for the plugin
	Repo *Repo
}

// Command formats and outputs the command necessary for
// Kaniko to build and publish a Docker image.
func (p *Plugin) Command() *exec.Cmd {
	logrus.Debug("creating kaniko command from plugin configuration")

	// variable to store flags for command
	var flags []string

	// iterate through all image build args
	for _, arg := range p.Image.Args {
		// add flag for build args from provided image build arg
		flags = append(flags, fmt.Sprintf("--build-arg=%s", arg))
	}

	// check if repo caching is enabled
	if p.Repo.Cache {
		// add flag for caching from provided repo cache
		flags = append(flags, fmt.Sprint("--cache"))

		// check if repo cache name is provided
		if len(p.Repo.CacheName) > 0 {
			// add flag for cache repo from provided repo cache name
			flags = append(flags, fmt.Sprintf("--cache-repo=%s", p.Repo.CacheName))
		} else {
			// add flag for cache repo from provided repo name
			flags = append(flags, fmt.Sprintf("--cache-repo=%s", p.Repo.Name))
		}
	}

	// add flag for context from provided image context
	flags = append(flags, fmt.Sprintf("--context=%s", p.Image.Context))

	// check if repo auto tagging is enabled
	if p.Repo.AutoTag {
		// check what build event was provided
		switch p.Build.Event {
		case "tag":
			// add build tag to list of repo tags
			p.Repo.Tags = append(p.Repo.Tags, p.Build.Tag)
		default:
			// add build sha to list of repo tags
			p.Repo.Tags = append(p.Repo.Tags, p.Build.Sha)
		}
	}

	// iterate through all repo tags
	for _, tag := range p.Repo.Tags {
		// add flag for tag from provided repo tag
		flags = append(flags, fmt.Sprintf("--destination=%s:%s", p.Repo.Name, tag))
	}

	// add flag for dockerfile from provided image dockerfile
	flags = append(flags, fmt.Sprintf("--dockerfile=%s", p.Image.Dockerfile))

	// check if registry dry run is enabled
	if p.Registry.DryRun {
		// add flag for building without publishing image
		flags = append(flags, fmt.Sprint("--no-push"))
	}

	// add flag for logging verbosity
	flags = append(flags, fmt.Sprintf("--verbosity=%s", logrus.GetLevel()))

	return exec.Command(kanikoBin, flags...)
}

// Exec formats and runs the commands for building and publishing a Docker image.
func (p *Plugin) Exec() error {
	logrus.Debug("running plugin with provided configuration")

	// create registry file for authentication
	err := p.Registry.Write()
	if err != nil {
		return err
	}

	// run kaniko command from plugin configuration
	err = execCmd(p.Command())
	if err != nil {
		return err
	}

	return nil
}

// Validate verifies the Plugin is properly configured.
func (p *Plugin) Validate() error {
	logrus.Debug("validating plugin configuration")

	// validate build configuration
	err := p.Build.Validate()
	if err != nil {
		return err
	}

	// validate image configuration
	err = p.Image.Validate()
	if err != nil {
		return err
	}

	// validate registry configuration
	err = p.Registry.Validate()
	if err != nil {
		return err
	}

	// validate repo configuration
	err = p.Repo.Validate()
	if err != nil {
		return err
	}

	return nil
}
