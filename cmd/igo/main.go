package main

import (
	"flag"
	"fmt"
	"igo/config"
	"igo/log"
	"igo/server"
	"os"
	"os/signal"
	"syscall"
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

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sc
		log.Warnf("Got signal [%d] to exit.", sig)
		srv.Close()
		os.Exit(0)
	}()

	log.Error(srv.Run())
}
