package config

import "testing"

func Test_ParseConfig(t *testing.T) {
	f := "./igo_config.toml"
	c, e := ParseConfig(f)
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("config:%+v", c)
}
