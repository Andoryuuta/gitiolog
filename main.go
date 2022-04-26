package main

import (
	"fmt"
	"time"
)

func main() {
	resolver := newResolver()
	output := make(chan ResolvedShortlink, 1024)
	go resolver.ResolveRange(0, 62*62*62*62-1, output)

	go func() {
		for {
			time.Sleep(10 * time.Second)
			fmt.Printf("RPS: %v\n", resolver.GetRPS())
		}
	}()

	for {
		resolved := <-output
		_ = resolved
		//fmt.Println(resolved)
	}
}
