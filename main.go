package main

import (
	"fmt"
	"github.com/attachmentgenie/atc/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	fmt.Printf("atc %s, commit %s, built at %s by %s\n\n", version, commit, date, builtBy)
	cmd.Execute()
}
