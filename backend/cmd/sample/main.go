package main

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"

	"github.com/creack/pty"
)

// this /sample/main.go is only for experimentation purpose.
// any code written here is not associated with the main project and will be deleted after the experimentation is done.
func main() {

	cmd := exec.Command("git", "clone", "--depth", "1", "file:///home/roshan-anand/workspace/personal/godploy_workspace/samples/portfolio", "./newapp-main-ikc", "&&", "git", "-C", "./newapp-main-ikc", "fetch", "--depth", "1", "origin", "main", "&&", "git", "-C", "./newapp-main-ikc", "switch", "-C", "deploy_branch", "FETCH_HEAD")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(ptmx)

	go func() {
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				fmt.Println(line)
			}

			if err != nil {
				break
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("finished job")
}
