package main

import (
	"github.com/Zaba505/backup/cmd"
)

func main() {
	cli := cmd.Build()

	cmd.CheckError(
		cmd.LogFatal,
		cli.Run(),
	)
}
