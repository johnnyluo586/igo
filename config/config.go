package config

import (
	"io/ioutil"
	"time"

	"github.com/BurntSushi/toml"
)

//Config all the config
type Config struct {
	Server ServerConfig `toml:"server"`
}

//ServerConfig the server config
type ServerConfig struct {
	Addr   string `toml:"addr"`
	User   string `toml:"user"`
	Passwd string `toml:"passwd"`
	Schema string `toml:"schema"`

	MaxClientConn int64         `toml:"maxClientConn"`
	WriteTimeout  time.Duration `toml:"writeTimeout"`
}

//ParseConfig parse Config from toml file path.
func ParseConfig(fname string) (*Config, error) {
	content, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	c := new(Config)
	if err := toml.Unmarshal(content, c); err != nil {
		return nil, err
	}
	return c, nil

}
