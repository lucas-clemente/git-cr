package main

import (
	"io"
	"log"
	"os"
)

func main() {
	f, err := os.Create("/Users/lucas/Desktop/log.txt")
	if err != nil {
		log.Fatal(err)
	}
	io.Copy(f, os.Stdin)
}
