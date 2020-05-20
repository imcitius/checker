package cmd

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"syscall"
)

var (
	// Used for flags.
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "go-boilerplate",
		Short: "Checker runner",
		Long:  `^_^`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config", "config file (default is ./config.json)")
	rootCmd.PersistentFlags().StringP("debugLevel", "D", "info", "Debug level: Debug,Info,Warn,Error,Fatal,Panic")
	rootCmd.PersistentFlags().Bool("viper", true, "use Viper for configuration")
	viper.BindPFlag("debugLevel", rootCmd.PersistentFlags().Lookup("debugLevel"))
	viper.BindPFlag("useViper", rootCmd.PersistentFlags().Lookup("viper"))
	viper.SetDefault("debugLevel", "INFO")

	rootCmd.AddCommand(testCfg)
	rootCmd.AddCommand(checkCommand)
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

func initConfig() {

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetDefault("HTTPPort", "80")
		viper.SetDefault("debugLevel", "Info")

		viper.SetConfigName("config")         // name of config file (without extension)
		viper.SetConfigType("json")           // REQUIRED if the config file does not have the extension in the name
		viper.AddConfigPath("/etc/appname/")  // path to look for the config file in
		viper.AddConfigPath("$HOME/.appname") // call multiple times to add many search paths
		viper.AddConfigPath(".")              // optionally look for config in the working directory
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	dl, err := logrus.ParseLevel(viper.GetString("debugLevel"))
	if err != nil {
		log.Panicf("Cannot parse debug level: %v", err)
	} else {
		log.SetLevel(dl)
	}

	err = Config.loadConfig()
	if err != nil {
		log.Infof("Config load error: %s", err)
	}

	err = Config.fillDefaults()
	if err != nil {
		panic(err)
	}
	Config.fillUUIDs()
	if err != nil {
		panic(err)
	}

	signalINT = make(chan os.Signal, 1)
	signalHUP = make(chan os.Signal, 1)
	doneCh = make(chan bool)
	schedulerSignalCh = make(chan bool)
	signal.Notify(signalINT, syscall.SIGINT)
	signal.Notify(signalHUP, syscall.SIGHUP)

}

var checkCommand = &cobra.Command{
	Use:   "check",
	Short: "Run scheduler and execute checks",
	Run:   mainChecker,
}

func mainChecker(cmd *cobra.Command, args []string) {
	initConfig()

	go signalWait()

	wg.Add(1)
	go Config.runScheduler(schedulerSignalCh, &wg)
	wg.Wait()

	if !interrupt {
		mainChecker(rootCmd, []string{})
	}
}

func signalWait() {

	select {
	case <-signalINT:
		log.Infof("Got SIGINT")
		interrupt = true
		schedulerSignalCh <- true
	case <-signalHUP:
		log.Infof("Got SIGHUP")
		schedulerSignalCh <- true
	}
}
