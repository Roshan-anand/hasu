package main

import (
	"fmt"
	"log"
	"net/url"
)

func main() {

	urls := []string{
		"example.com",
		"example.com/path/to/resource",
		"https://example.com/",
	}

	for _, u := range urls {
		pUrl, err := url.Parse(u)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%q\n", u)
		fmt.Printf("Scheme: %q\n", pUrl.Scheme)
		fmt.Printf("Host:   %q\n", pUrl.Host)
		fmt.Printf("Path:   %q\n\n", pUrl.Path)
	}
}
