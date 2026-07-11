package main

import (
	"fmt"
	"strings"
)

// this /sample/main.go is only for experimentation purpose.
// any code written here is not associated with the main project and will be deleted after the experimentation is done.
func main() {

	bodys := []string{"/godploy deploy", "", "good morning", "/godploy not", "/godploy deploy myself", "has deploy"}

	for _, body := range bodys {
		cmd := strings.Split(strings.TrimSpace(body), " ")
		len := len(cmd)
		if len < 2 || cmd[0] != "/godploy" || cmd[1] != "deploy" {
			fmt.Println("not valid cmd :", body)
		} else {
			fmt.Println("valid cmd :", body)
		}
	}
}
