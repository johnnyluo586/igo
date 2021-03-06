package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
)

import (
	"igo/config"
	"igo/log"
	"igo/server"
)

var (
	banner = ` 
        Welcome IGO!
`

	configFile = flag.String("config", "./igo_config.toml", "Input the config file path")
)

func main() {
	//print banner
	fmt.Println(banner)

	go func() {
		log.Error(http.ListenAndServe(":6060", nil))
	}()

	//load config
	flag.Parse()
	var cfg *config.Config
	if configFile != nil {
		c, err := config.ParseConfig(*configFile)
		if err != nil {
			log.Error(err)
			os.Exit(-1)
		}
		cfg = c
	}

	//new and run server
	log.Error(server.NewServer(cfg).Run())
}
