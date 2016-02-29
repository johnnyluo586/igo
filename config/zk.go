package config

import (
	"igo/log"
	"time"

	"encoding/json"

	"github.com/BurntSushi/toml"
	"github.com/samuel/go-zookeeper/zk"
)

//ZKConfig zk config
type ZKConfig struct {
	Addrs   []string `toml:"zk_addrs"`
	Path    string   `toml:"zk_path"`
	Timeout int64    `toml:"zk_timeout"`
	Data    struct {
		Content  string `json:"content"`
		Metadata struct {
			ContentType string `json:"content_type"`
			Deleted     int    `json:"deleted"`
			Mtime       int    `json:"mtime"`
			Ptime       int    `json:"ptime"`
			State       int    `json:"state"`
		} `json:"metadata"`
	}
}

var _ Configer = &ZKConfig{}

//Parse parse config from zk.
func (z *ZKConfig) Parse() (*Config, error) {
	cli, _, err := zk.Connect(z.Addrs, time.Duration(z.Timeout)*time.Second)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	data, _, err := cli.Get(z.Path)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	log.Infof("Load ZK Config: %+v", z.Path)

	if err := json.Unmarshal(data, &z.Data); err != nil {
		log.Error(err)
		return nil, err
	}
	log.Debugf("\n%v", string(z.Data.Content))

	c := new(Config)
	if err := toml.Unmarshal([]byte(z.Data.Content), c); err != nil {
		log.Error(err)
		return nil, err
	}
	return c, nil
}
