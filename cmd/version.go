package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"my/checker/config"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Checker",
	Long:  `All software has versions. This is Hugo's (no, Checker's of course ;)`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Checker version: " + config.Version + "" + config.VersionBuild)
	},
}
