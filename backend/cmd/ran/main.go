package main

import (
	"fmt"
	"net/url"
)

func main() {
	link := "https://github.com/Roshan-anand/code-join.git"

	u, err := url.Parse(link)
	if err != nil {
		panic(err)
	}

	// Remove leading "/"
	path := u.Host + u.Path

	fmt.Println(path)
}
