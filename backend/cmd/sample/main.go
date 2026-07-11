package main

import (
	"fmt"
	"net/url"
	"strings"
)

// this /sample/main.go is only for experimentation purpose.
// any code written here is not associated with the main project and will be deleted after the experimentation is done.
func main() {

	urls := []string{"https://example.com", "example.com", "http://example.com/xyz", "noturl", "www.example.com", "https://www.example.com"}

	for _, u := range urls {
		fmt.Println("-------------------------------------")
		fmt.Println("URL:", u)

		if !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
			u = "https://" + u
		}

		fmt.Println("after validaton :", u)

		parseUrl, err := url.Parse(u)
		if err != nil {
			fmt.Println("Error parsing URL : ", err)
			continue
		}
		fmt.Println("schema :", parseUrl.Scheme)
		fmt.Println("host :", parseUrl.Host)
		fmt.Println("hostname :", parseUrl.Hostname())
		fmt.Println("path:", parseUrl.Path)
	}

}
