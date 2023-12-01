// chat.go
package main

import (
	"flag"
	"fmt"
	"log"
	"v1/node"
)

func main() {
	relay := flag.String("relay", "", "relay addrs")
	flag.Parse()

	if *relay == "" {
		fmt.Println("Please Use -relay ")
		return
	}
	flag.Parse()

	n := node.NewNode(*relay)
	if n == nil {
		return
	}

	if err := n.ConnectRelay(*relay); err != nil {
		log.Println(err.Error())
		return
	}

	log.Println("Connect relay success Next Login...")
	n.Login()

	log.Println("Im ", n.ID())
	select {
	case <-n.Ctx.Done():
		log.Println(n.Ctx.Err().Error())
	}
}
