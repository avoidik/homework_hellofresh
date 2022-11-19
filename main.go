package main

import (
	"os"
)

// tagRelease allows to set release version at compilation time
var tagRelease string

// app entrypoint
func main() {
	os.Exit(startServer())
}
