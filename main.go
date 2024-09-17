package main

import (
	"os"

	"github.com/carlmjohnson/exitcode"
	"github.com/earthboundkid/kawatte/replaceall"
)

func main() {
	exitcode.Exit(replaceall.CLI(os.Args[1:]))
}
