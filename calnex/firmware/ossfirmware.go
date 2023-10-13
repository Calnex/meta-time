/*
Copyright (c) Facebook, Inc. and its affiliates.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package firmware

import (
	"path/filepath"
	"strings"

	version "github.com/hashicorp/go-version"
)

// OSSFW is an open source implementation of the FW interface
type OSSFW struct {
	Filepath string
}

// Version downloads latest firmware version
// sentinel_fw_v2.13.1.0.5583D-20210924.tar
// sentinel_fw_R18.1.0.tar
// calnex_combined_fw_R18.1.0.tar
func (f *OSSFW) Version() (*version.Version, error) {
	var err error
	var vs string
	var v *version.Version

	basename := filepath.Base(f.Filepath)

	if strings.HasPrefix(basename, "sentinel") {
		vs = strings.ReplaceAll(strings.TrimSuffix(basename, filepath.Ext(basename)), "sentinel_fw_", "")
	} else if strings.HasPrefix(basename, "sentry") {
		vs = strings.ReplaceAll(strings.TrimSuffix(basename, filepath.Ext(basename)), "sentry_fw_", "")
	} else if strings.HasPrefix(basename, "calnex") {
		vs = strings.ReplaceAll(strings.TrimSuffix(basename, filepath.Ext(basename)), "calnex_combined_fw_", "")
	} else {
		vs = "v2.13.1.0.5583d-20210924"
		//fw := OSSFW{
		//	Filepath: expectedFilePath,
		//}
		//v, err := fw.Version()
	}

	if strings.HasPrefix(vs, "v") {
		v, err = version.NewVersion(strings.SplitN(strings.ToLower(vs), ".", 2)[1])
	} else if strings.HasPrefix(vs, "R") {
		v, err = version.NewVersion(strings.TrimPrefix(strings.ToLower(vs), "r"))
	}
	return v, err
}

// Path downloads latest firmware version
func (f *OSSFW) Path() (string, error) {
	return f.Filepath, nil
}
