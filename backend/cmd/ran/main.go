package main

import (
	"fmt"
	"net"
	"net/url"
)

func main() {
	link := "https://coaching-anyone-badly-trader.trycloudflare.com"
	u, err := url.Parse(link)
	if err != nil {
		panic(err)
	}

	addrs, err := net.LookupHost(u.Host)
	if err != nil {
		panic(err)
	}

	fmt.Println(addrs)
}
