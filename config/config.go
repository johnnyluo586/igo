package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

//Config all the config
type Config struct {
	Server ServerConfig `yaml:"server"`
}

//ServerConfig the server config
type ServerConfig struct {
	Addr          string `yaml:"addr"`
	MaxClientConn int    `yaml:"maxClientConn"`
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
