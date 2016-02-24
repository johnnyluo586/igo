package main

import (
	"fmt"
	"igo/server"
)

func main() {
	fmt.Println("welcome igo!")
	srv := server.NewServer()
	srv.Run()
}
