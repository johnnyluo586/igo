package config

import (
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

//Configer the config interface
type Configer interface {
	Parse() (*Config, error)
}

//Config all the config
type Config struct {
	Server ServerConfig `toml:"Server"`
}

//ServerConfig the server config
type ServerConfig struct {
	Addr   string `toml:"addr"`
	Schema string `toml:"schema"`
	User   string `toml:"user"`
	Passwd string `toml:"passwd"`

	MaxClient    int64 `toml:"maxClient"`
	WriteTimeout int64 `toml:"writeTimeout"`
	ReadTImeout  int64 `toml:"readTimeout"`

	MaxIdleConn int `toml:"maxIdleConn"`
	MaxConnNum  int `toml:"maxConnNum"`
}

//ParseConfig parse Config from toml file path.
func ParseConfig(fname string) (*Config, error) {
	content, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	cfg := ZKConfig{}
	if err := toml.Unmarshal(content, &cfg); err != nil {
		return nil, err
	}

	return cfg.Parse()
}
