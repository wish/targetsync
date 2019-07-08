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

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Config for the targetsync
type Config struct {
	ConsulConfig       `yaml:"consul"`
	AWSConfig          `yaml:"aws"`
	K8sEndpointsConfig `yaml:"k8s_enpoints"`

	SyncConfig `yaml:"syncer"`
}

func (c *Config) Validate() error {
	return c.SyncConfig.Validate()
}

// ConsulConfig holds the configuration for the consul source
type ConsulConfig struct {
	ClientConfig *consulApi.Config `yaml:"client"`
	ServiceName  string            `yaml:"service_name"`
	Tag          string            `yaml:"tag"`
}

// AWSConfig holds the configuration for the aws destination
type AWSConfig struct {
	TargetGroupARN   string `yaml:"target_group_arn"`
	AvailabilityZone string `yaml:"availability_zone"`
}

type K8sConfig struct {
	InCluster      bool   `yaml:"in_cluster"`
	KubeConfigPath string `yaml:"kubeconfig_path"`
}

type K8sEndpointsConfig struct {
	K8sConfig `yaml:"k8s"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Port      int    `yaml:"port"`
}

// SyncConfig holds options for the Syncer
type SyncConfig struct {
	LockOptions `yaml:"lock_options"`

	RemoveDelay time.Duration `yaml:"remove_delay"`
}

func (c SyncConfig) Validate() error {
	if c.LockOptions.TTL <= time.Duration(0) {
		return fmt.Errorf("TTL for locks must be >0")
	}
	return nil
}
