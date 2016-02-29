package config

import "testing"

func Test_Parse(t *testing.T) {
	zk := ZKConfig{
		Addrs:   []string{"192.168.69.6:2181", "192.168.69.6:2182"},
		Path:    "/path",
		Timeout: 10,
	}
	data, err := zk.Parse()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("config:%+v", data)
}
