package main

import (
	"log"
	"os/exec"
)

func main() {

	cmd := exec.Command("bash", "-c", `echo "one"`)

	cmd.Args = append(cmd.Args, "&&", `echo "two"`)

	res, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Error running command: %v", err)
	}
	log.Printf("Command output: %s", string(res))

}
