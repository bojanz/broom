package broom

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v2"
)

// Config represents Broom's configuration.
type Config map[string]ProfileConfig

// Profiles returns a list of all configured profiles.
func (c Config) Profiles() []string {
	profiles := make([]string, 0, len(c))
	for profile := range c {
		profiles = append(profiles, profile)
	}
	sort.Strings(profiles)

	return profiles
}

// ProfileConfig represents Broom's per-profile configuration.
type ProfileConfig struct {
	SpecFile  string `yaml:"spec_file"`
	ServerURL string `yaml:"server_url"`
	Token     string `yaml:"token"`
	TokenCmd  string `yaml:"token_cmd"`
}

// ReadConfig reads a config file with the given filename.
func ReadConfig(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	if len(data) == 0 {
		return Config{}, fmt.Errorf("%s is empty", filename)
	}
	config := Config{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

// WriteConfig writes the given config to the given filename.
func WriteConfig(filename string, cfg Config) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return nil
	}
	err = os.WriteFile(filename, b, 0666)
	if err != nil {
		return err
	}

	return nil
}
