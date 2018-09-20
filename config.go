package targetsync

import (
	"fmt"
	"io/ioutil"
	"time"

	consulApi "github.com/hashicorp/consul/api"

	yaml "gopkg.in/yaml.v2"
)

// ConfigFromFile Loads a config file from `path`
func ConfigFromFile(path string) (*Config, error) {
	// load the config file
	cfg := &Config{
		ConsulConfig: ConsulConfig{
			ClientConfig: consulApi.DefaultConfig(),
		},
	}
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error loading config: %v", err)
	}
	err = yaml.Unmarshal([]byte(configBytes), &cfg)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshaling config: %v", err)
	}

	return cfg, nil
}

// Config for the targetsync
type Config struct {
	ConsulConfig `yaml:"consul"`
	AWSConfig    `yaml:"aws"`

	SyncConfig `yaml:"syncer"`
}

// ConsulConfig holds the configuration for the consul source
type ConsulConfig struct {
	ClientConfig *consulApi.Config `yaml:"client"`
	ServiceName  string            `yaml:"service_name"`
}

// AWSConfig holds the configuration for the aws destination
type AWSConfig struct {
	TargetGroupARN string `yaml:"target_group_arn"`
}

// SyncConfig holds options for the Syncer
type SyncConfig struct {
	LockOptions `yaml:"lock_options"`

	RemoveDelay time.Duration `yaml:"remove_delay"`
}
