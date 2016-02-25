package main

import (
	"flag"
	"fmt"
	"igo/config"
	"igo/log"
	"igo/server"
	"os"
)

var (
	banner = ` 
        Welcome IGO!
`

	configFile = flag.String("config", "./igo.yaml", "Input the config file path")
)

func main() {
	//print banner
	fmt.Println(banner)

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
	srv := server.NewServer(cfg)
	srv.Run()
}
