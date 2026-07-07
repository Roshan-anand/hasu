package main

import (
	"context"
	"fmt"
	"time"
)

// this /sample/main.go is only for experimentation purpose.
// any code written here is not associated with the main project and will be deleted after the experimentation is done.
func main() {
	ctx, cancle := context.WithCancel(context.Background())

	go func() {
		time.Sleep(1 * time.Second)
		cancle()
	}()

	<-ctx.Done()
	fmt.Println("Context done ", ctx.Err())
}
