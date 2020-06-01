package main

import (
	"fmt"
	"my/checker/cmd"
)

var Version string
var VersionSHA string
var VersionBuild string

func main() {
	if Version != "" && VersionSHA != "" && VersionBuild != "" {
		fmt.Printf("Start %s (commit: %s; build: %s)\n", Version, VersionSHA, VersionBuild)
	} else {
		fmt.Println("Start dev ")
	}

	cmd.Execute()
}
