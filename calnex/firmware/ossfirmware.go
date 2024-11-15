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
	"fmt"
	"path/filepath"
	"strings"

	version "github.com/hashicorp/go-version"
)

// OSSFW is an open source implementation of the FW interface
type OSSFW struct {
	filepath string
	version  *version.Version
}

// NewOSSFW returns initialized version of OSSFW
func NewOSSFW(source string) (*OSSFW, error) {
	var err error
	var vs string
	var v *version.Version

	fw := &OSSFW{
		filepath: source,
	}
	basename := filepath.Base(fw.filepath)
	// Extract version from filename
	// sentinel_fw_v2.13.1.0.5583D-20210924.tar
	// calnex_combined_fw_R21.0.0.9705-20241111.tar
	if strings.HasPrefix(basename, "sentinel_fw_") {
		vs = strings.ReplaceAll(strings.TrimSuffix(basename, filepath.Ext(basename)), "sentinel_fw_", "")
	} else if strings.HasPrefix(basename, "sentry_fw_") {
		vs = strings.ReplaceAll(strings.TrimSuffix(basename, filepath.Ext(basename)), "sentry_fw_", "")
	} else if strings.HasPrefix(basename, "calnex_combined_fw_") {
		vs = strings.ReplaceAll(strings.TrimSuffix(basename, filepath.Ext(basename)), "calnex_combined_fw_", "")
	} else {
		return nil, fmt.Errorf("Unexpected file string, expected sentinel_fw_ sentry_fw_ or calnex_combined_fw_  at start: %s", basename)
	}

	if strings.HasPrefix(vs, "v") {
		v, err = version.NewVersion(strings.SplitN(strings.ToLower(vs), ".", 2)[1])
	} else if strings.HasPrefix(vs, "R") {
		v, err = version.NewVersion(strings.TrimPrefix(strings.ToLower(vs), "r"))
	} else {
		return nil, fmt.Errorf("Unexpected file string, expected v2. or R. : %s", vs)
	}
	fw.version = v
	return fw, err
}

// Version downloads latest firmware version
// sentinel_fw_v2.13.1.0.5583D-20210924.tar
func (f *OSSFW) Version() *version.Version {
	return f.version
}

// Path downloads latest firmware version
func (f *OSSFW) Path() (string, error) {
	return f.filepath, nil
}
