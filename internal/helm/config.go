// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package helm

import "path/filepath"

type Config struct {
	Tiller struct {
		Version string
	}

	Home string

	Repositories map[string]string

	// flag signaling if helm3 is enabled or not
	V3 bool
}

// Validate validates the configuration.
func (c Config) Validate() error {
	return nil
}

func (c Config) GetPath(organizationName string) string {
	return filepath.Join(c.Home, organizationName)
}
