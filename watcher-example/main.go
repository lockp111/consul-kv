package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	kv "github.com/lockp111/consul-kv"
)

func main() {
	var err error
	cli := kv.NewConfig(kv.WithPrefix("kvTest"))
	err = cli.Connect()
	if err != nil {
		log.Fatalln(err)
	}

	err = cli.Watch("test2", func(r *kv.Result) {
		log.Printf("new value: %s", r.String())
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = cli.Watch("test1", func(r *kv.Result) {
		log.Printf("new value: %s", r.String())
	})
	if err != nil {
		log.Fatalln(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		log.Printf("exit with signal %s", s.String())

		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			cli.StopWatch("test1", "test3", "test2")
			time.Sleep(time.Second * 2)
			close(c)
			return
		case syscall.SIGHUP:
		default:
			close(c)
			return
		}
	}
}
