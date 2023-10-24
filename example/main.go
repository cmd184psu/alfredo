package main

import (
	"fmt"

	"github.com/cmd184psu/alfredo"
)

func main() {

	alfredo.Touch("afile.txt")
	alfredo.System3("ls -lah")
	fmt.Println("hello")
	alfredo.VerbosePrintln("alfredo- don't see me?")
	alfredo.SetVerbose(true)
	alfredo.VerbosePrintln("alfredo- see me?")
}
