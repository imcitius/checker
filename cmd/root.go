package cmd

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"my/checker/config"
	"my/checker/scheduler"
	"my/checker/web"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	// Used for flags.
	cfgFile string

	rootCmd = &cobra.Command{
		Use:   "checker",
		Short: "Checker runner",
		Long:  `^_^`,
	}
	signalINT, signalHUP                   chan os.Signal
	doneCh, schedulerSignalCh, webSignalCh chan bool
	wg                                     sync.WaitGroup
	interrupt                              bool
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

	signalINT = make(chan os.Signal)
	signalHUP = make(chan os.Signal)
	doneCh = make(chan bool)
	schedulerSignalCh = make(chan bool)
	webSignalCh = make(chan bool)
	signal.Notify(signalINT, syscall.SIGINT)
	signal.Notify(signalHUP, syscall.SIGHUP)
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

func initConfig() {

	config.Log.Debug("initConfig: load config file")

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
		config.Log.Panicf("Cannot parse debug level: %v", err)
	} else {
		config.Log.SetLevel(dl)
	}

	err = config.LoadConfig()
	if err != nil {
		config.Log.Infof("Config load error: %s", err)
	}

	err = config.FillDefaults()
	if err != nil {
		panic(err)
	}
	err = config.FillUUIDs()
	if err != nil {
		panic(err)
	}
	err = config.FillTimeouts()
	if err != nil {
		panic(err)
	}
}

var checkCommand = &cobra.Command{
	Use:   "check",
	Short: "Run scheduler and execute checks",
	Run:   mainChecker,
}

func mainChecker(cmd *cobra.Command, args []string) {
	for {
		logrus.Info("Start main loop")
		interrupt = false
		initConfig()

		go signalWait()

		wg.Add(2)
		go scheduler.RunScheduler(schedulerSignalCh, &wg)
		go web.WebInterface(webSignalCh, &wg)
		wg.Wait()

		if interrupt {
			os.Exit(1)
		}
	}
}

func signalWait() {
	for {
		select {
		case <-signalINT:
			config.Log.Infof("Got SIGINT")
			interrupt = true
			schedulerSignalCh <- true
			webSignalCh <- true
		case <-signalHUP:
			config.Log.Infof("Got SIGHUP")
			//schedulerSignalCh <- true
			//webSignalCh <- true
			initConfig()
		}
	}
}

var testCfg = &cobra.Command{
	Use:   "testcfg",
	Short: "unmarshal config file into config structure",
	Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {

		config.Log.Infof("Config :\n%+v\n\n\n", config.Config)
	},
}
