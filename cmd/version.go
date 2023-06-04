package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version number of this cli tool",
	Long:  "The version number is stored in a config file, loaded by viper. This command will print that version number out",
	Run:   ShowVersion,
}

// ShowVersion prints out the cli version number
func ShowVersion(cmd *cobra.Command, args []string) {
	showVersion()
}

// Dumb version of ShowVersion(). Used for testing
func showVersion() {
	fmt.Printf("go-cli-boilerplate version: %s \n", viper.GetString("version"))
}
