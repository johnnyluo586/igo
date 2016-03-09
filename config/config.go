package config

import (
	"io/ioutil"

	"time"

	"github.com/BurntSushi/toml"
)

//Configer the config interface
type Configer interface {
	Parse() (*Config, error)
}

//Config all the config
type Config struct {
	Server ServerConfig `toml:"Server"`
	// Redis  ServerConfig `toml:"Server.redis"`
}

//ServerConfig the server config
type ServerConfig struct {
	Listen    string `toml:"listen"`
	Addr      string `toml:"dbaddr"`
	DBName    string `toml:"dbname"`
	User      string `toml:"user"`
	Passwd    string `toml:"passwd"`
	Collation string `toml:"collation"`

	MaxClient    int64 `toml:"maxClient"`
	WriteTimeout int   `toml:"writeTimeout"`
	ReadTimeout  int   `toml:"readTimeout"`
	MaxLifeTime  int   `toml:"maxLifeTmie"`
	MaxIdleConn  int   `toml:"maxIdleConn"`
	MaxConnNum   int   `toml:"maxConnNum"`

	Loc              *time.Location //location time
	ColumnsWithAlias bool

	Strict bool
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
