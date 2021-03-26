package main

import (
	"log"

	_ "tcw.im/rtfd/assets"
	"tcw.im/rtfd/cmd"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	cmd.Execute()
}
