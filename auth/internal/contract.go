// SPDX-License-Identifier: ice License 1.0

package internal

// Private API.

type (
	config struct {
		WintrServerAuth struct {
			Credentials struct {
				FilePath    string `yaml:"filePath"`
				FileContent string `yaml:"fileContent"`
			} `yaml:"credentials" mapstructure:"credentials"`
		} `yaml:"wintr/server/auth" mapstructure:"wintr/server/auth"` //nolint:tagliatelle // Nope.
	}
)
