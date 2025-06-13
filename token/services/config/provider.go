/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	viperutil "github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config/viper"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var logOutput = os.Stderr

type provider struct {
	v *viper.Viper
}

func NewProvider(raw []byte) (*provider, error) {
	p := &provider{}
	if err := p.load(raw); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *provider) GetDuration(key string) time.Duration {
	return p.v.GetDuration(key)
}

func (p *provider) GetBool(key string) bool {
	return p.v.GetBool(key)
}

func (p *provider) GetInt(key string) int {
	return p.v.GetInt(key)
}

func (p *provider) GetStringSlice(key string) []string {
	return p.v.GetStringSlice(key)
}

func (p *provider) AddDecodeHook(f driver.DecodeHookFuncType) error {
	return nil
}

func (p *provider) UnmarshalKey(key string, rawVal interface{}) error {
	return viperutil.EnhancedExactUnmarshal(p.v, key, rawVal)
}

func (p *provider) IsSet(key string) bool {
	return p.v.IsSet(key)
}

func (p *provider) GetPath(key string) string {
	path := p.v.GetString(key)
	if path == "" {
		return ""
	}

	return TranslatePath(filepath.Dir(p.v.ConfigFileUsed()), path)
}

func (p *provider) TranslatePath(path string) string {
	if path == "" {
		return ""
	}

	return TranslatePath(filepath.Dir(p.v.ConfigFileUsed()), path)
}

func (p *provider) GetString(key string) string {
	return p.v.GetString(key)
}

func (p *provider) ConfigFileUsed() string {
	return p.v.ConfigFileUsed()
}

func (p *provider) load(raw []byte) error {
	p.v = viper.New()

	err := p.v.ReadConfig(bytes.NewBuffer(raw))
	if err != nil {
		return errors.Wrap(err, "failed to load config file")
	}

	if err := p.substituteEnv(); err != nil {
		return err
	}

	logging.Init(logging.Config{
		Format:  p.v.GetString("logging.format"),
		Writer:  logOutput,
		LogSpec: p.v.GetString("logging.spec"),
	})

	return nil
}

// Manually override keys if the respective environment variable is set, because viper doesn't do
// that for UnmarshalKey values (see https://github.com/spf13/viper/pull/1699).
// Example: CORE_LOGGING_FORMAT sets logging.format.
func (p *provider) substituteEnv() error {
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, strings.ToUpper(config.CmdRoot)+"_") {
			continue
		}

		env := strings.Split(e, "=")
		val := env[1]
		if len(val) == 0 {
			continue
		}
		key, val := env[0], strings.Join(env[1:], "=")

		noprefix := strings.TrimLeft(key, strings.ToUpper(config.CmdRoot)+"_")
		key = strings.ToLower(strings.ReplaceAll(noprefix, "_", "."))

		// nested key
		keys := strings.Split(key, ".")
		parent := strings.Join(keys[:len(keys)-1], ".")
		if !p.v.IsSet(parent) {
			fmt.Println("applying " + env[0] + " - parent not found in core.yaml: " + parent)
			p.v.Set(key, val)
			continue
		}

		k := p.v.GetStringMap(key)
		if len(k) > 0 {
			fmt.Println("-- skipping " + env[0] + ": cannot override maps")
			continue
		}

		root := p.v.GetStringMap(keys[0])
		if err := setDeepValue(root, keys, val); err != nil {
			return errors.Wrap(err, "error when substituting")
		}
		p.v.Set(keys[0], root)
		fmt.Println("applying " + env[0])
	}
	return nil
}

// Function to set the value at the deepest level
func setDeepValue(m map[string]any, keys []string, value any) error {
	// key = root but we don't have the map by reference
	if len(keys) < 2 {
		return errors.New("can't set root key")
	}

	current := m
	// traverse to the last map
	for i := 1; i < len(keys)-1; i++ {
		key := keys[i]
		nextMap, ok := current[key].(map[string]any)
		if !ok {
			return errors.New("expected map at key " + key)
		}
		current = nextMap
	}
	lastKey := keys[len(keys)-1]
	current[lastKey] = value

	return nil
}

func TranslatePath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}

	return filepath.Join(base, p)
}
