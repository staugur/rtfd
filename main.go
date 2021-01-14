//go:generate statik -src=assets -include=*.cfg,*.js,*.sh -p assets -f
//go:generate mv assets/statik.go assets/assets.go
//go:generate go fmt assets/assets.go

package main

import (
	_ "tcw.im/rtfd/assets"
	"tcw.im/rtfd/cmd"
)

func main() {
	cmd.Execute()
}
