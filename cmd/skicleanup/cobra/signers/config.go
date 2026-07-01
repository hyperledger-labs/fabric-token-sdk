/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package signers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the database connection parameters loaded from the YAML config file.
type Config struct {
	// Driver is the database driver to use: "sqlite" or "postgres".
	Driver string `mapstructure:"driver"`
	// DataSource is the DSN / file path for the target database.
	DataSource string `mapstructure:"dataSource"`
	// TablePrefix is the optional prefix used when deriving table names.
	TablePrefix string `mapstructure:"tablePrefix"`
}

// LoadConfig reads the YAML config file at the given path and returns a Config.
func LoadConfig(path string) (Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.Driver != "sqlite" && cfg.Driver != "postgres" {
		return Config{}, fmt.Errorf("unsupported driver %q: must be \"sqlite\" or \"postgres\"", cfg.Driver)
	}
	if cfg.DataSource == "" {
		return Config{}, errors.New("dataSource must not be empty")
	}

	return cfg, nil
}
