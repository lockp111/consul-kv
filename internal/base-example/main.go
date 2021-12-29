package main

import (
	"fmt"
	"log"

	kv "github.com/lockp111/consul-kv"
)

func main() {
	var err error
	cli := kv.NewConfig(kv.WithAddress("http://127.0.0.1:8500"))
	err = cli.Init()
	if err != nil {
		log.Fatalln(err)
	}

	err = cli.Put("test1", "345")
	if err != nil {
		log.Fatalln(err)
	}

	ret := cli.Get("test1")
	if ret.Err() != nil {
		log.Fatalln(ret.Err())
	}

	fmt.Println(ret.String())
}
