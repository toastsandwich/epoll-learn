package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	server "github.com/toastsandwich/epoll-learn/http1.0_server"
	"golang.org/x/sys/unix"
)

//TODO:
// 1. Setup a buffer pool
// 2. Setup starter server
// 3. Design a worker pool

func main() {
	s, err := server.NewHTTPServer(&server.HTTPServerOpts{
		Addr: "0.0.0.0",
		Port: 8080,

		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, unix.SIGINT, unix.SIGTERM)

	go func() {
		if err := s.ListenAndServe(); err != nil {
			fmt.Println(err)
			return
		}
	}()
	<-sigC
	s.Close()
}
