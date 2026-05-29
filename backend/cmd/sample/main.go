package main

import (
	"log"
	"net/url"
)

func main() {
	fullurl := "github.com/Roshan-anand/godploy"
	u, err := url.Parse(fullurl)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Host: %s", u.Host)
	log.Printf("Path: %s", u.Path)
	log.Printf("full url: %s", fullurl)
}
