package cmd

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"my/checker/config"
	"my/checker/scheduler"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Viper config location
	cfgFile           string
	DefaultDebugLevel = "info"
	Config            = &config.Config{}
)

var rootCmd = &cobra.Command{
	Use: "checker",
	//Short: "This is a Cobra/Viper boilerplate",
	//Long:  `This is a Cobra/Viper boilerplate program written by patmizi in Go.`,

	//PersistentPreRun: func(cmd *cobra.Command, args []string) {
	//	fmt.Printf("Inside rootCmd PersistentPreRun with args: %v\n", args)
	//},

	PreRun: func(cmd *cobra.Command, args []string) {
		level, err := logrus.ParseLevel(DefaultDebugLevel)
		if err != nil {
			config.Log.Fatal(err)
		}
		config.Log.SetLevel(level)

		err = viper.Unmarshal(Config)
		if err != nil {
			fmt.Printf("unable to decode into config struct, %v", err)
		}
	},

	Run: func(cmd *cobra.Command, args []string) {
		scheduler.RunScheduler(&config.Log, Config)
	},

	//PostRun: func(cmd *cobra.Command, args []string) {
	//	fmt.Printf("Inside rootCmd PostRun with args: %v\n", args)
	//},
	//PersistentPostRun: func(cmd *cobra.Command, args []string) {
	//	fmt.Printf("Inside rootCmd PersistentPostRun with args: %v\n", args)
	//},
}

// Execute will execute the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", ".config.yaml", "config file location (defaults to .config.yaml)")
	cobra.OnInitialize(Config.InitConfig(cfgFile))
}
