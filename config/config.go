package config

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

//Config all the config
type Config struct {
	Server ServerConfig `yaml:"server"`
}

//ServerConfig the server config
type ServerConfig struct {
	Addr   string `yaml:"addr"`
	User   string `yaml:"user"`
	Passwd string `yaml:"passwd"`
	Schema string `yaml:"schema"`

	MaxClientConn int64         `yaml:"maxClientConn"`
	WriteTimeout  time.Duration `yaml:"writeTimeout"`
}

//ParseConfig parse Config from yaml file path.
func ParseConfig(fname string) (*Config, error) {
	content, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	c := new(Config)
	if err := yaml.Unmarshal(content, c); err != nil {
		return nil, err
	}
	return c, nil

}
