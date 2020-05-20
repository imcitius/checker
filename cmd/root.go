package cmd

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go-boilerplate/config"
	"go-boilerplate/web"
	"os"
	"os/signal"
	"sync"
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
	signalINT, signalHUP                   chan os.Signal
	doneCh, schedulerSignalCh, webSignalCh chan bool
	wg                                     sync.WaitGroup
	interrupt                              bool = false
	log                                         = logrus.New()
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

	err = config.LoadConfig()
	if err != nil {
		log.Infof("Config load error: %s", err)
	}

	err = config.FillDefaults()
	if err != nil {
		panic(err)
	}
	config.FillUUIDs()
	if err != nil {
		panic(err)
	}

	signalINT = make(chan os.Signal, 1)
	signalHUP = make(chan os.Signal, 1)
	doneCh = make(chan bool)
	schedulerSignalCh = make(chan bool)
	webSignalCh = make(chan bool)
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
	go runScheduler(schedulerSignalCh, &wg)
	go web.WebInterface(log, &wg, webSignalCh)
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

var testCfg = &cobra.Command{
	Use:   "testcfg",
	Short: "unmarshal config file into config structure",
	Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {

		log.Infof("Config :\n%+v\n\n\n", config.Config)
	},
}
