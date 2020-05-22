package cmd

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/semaphore"
	"my/checker/alerts"
	"my/checker/config"
	"my/checker/metrics"
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
	signalINT, signalHUP                                 chan os.Signal
	doneCh, schedulerSignalCh, botsSignalCh, webSignalCh chan bool
	wg                                                   sync.WaitGroup
	interrupt                                            bool
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config", "config file (default is ./config.json)")
	rootCmd.PersistentFlags().StringP("debugLevel", "D", "info", "Debug level: Debug,Info,Warn,Error,Fatal,Panic")
	rootCmd.PersistentFlags().Bool("bots", true, "start listening messenger bots")
	viper.BindPFlag("debugLevel", rootCmd.PersistentFlags().Lookup("debugLevel"))
	viper.BindPFlag("botsEnabled", rootCmd.PersistentFlags().Lookup("bots"))
	//viper.SetDefault("botsEnabled", true)
	//viper.SetDefault("debugLevel", "INFO")

	rootCmd.AddCommand(testCfg)
	rootCmd.AddCommand(checkCommand)

	signalINT = make(chan os.Signal)
	signalHUP = make(chan os.Signal)
	doneCh = make(chan bool)
	schedulerSignalCh = make(chan bool)
	webSignalCh = make(chan bool)
	botsSignalCh = make(chan bool)
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
	err = metrics.InitMetrics()
	if err != nil {
		panic(err)
	}

}

var checkCommand = &cobra.Command{
	Use:   "check",
	Short: "Run scheduler and execute checks",
	Run: func(cmd *cobra.Command, args []string) {
		mainChecker()
	},
}

func mainChecker() {
	// semaphore for web server port listening routine
	var sem = semaphore.NewWeighted(int64(1))

	for {
		config.Log.Info("Start main loop")
		interrupt = false
		initConfig()

		go signalWait()

		if sem.TryAcquire(1) {
			config.Log.Debugf("Fire webserver")
			go web.WebInterface(webSignalCh, sem)
		} else {
			config.Log.Debugf("Webserver already running")
		}

		wg.Add(1)
		go scheduler.RunScheduler(schedulerSignalCh, &wg)

		if viper.GetBool("botsEnabled") {
			config.Log.Debugf("botsEnabled is %v", viper.GetBool("botsEnabled"))
			wg.Add(1)
			alerts.InitBots(botsSignalCh, &wg)
		}

		wg.Wait()

		if interrupt {
			os.Exit(1)
		}
	}
}

func signalWait() {
	select {
	case <-signalINT:
		config.Log.Infof("Got SIGINT")
		interrupt = true
		if viper.GetBool("botsEnabled") {
			botsSignalCh <- true
		}
		schedulerSignalCh <- true
		//webSignalCh <- true
	case <-signalHUP:
		config.Log.Infof("Got SIGHUP")
		schedulerSignalCh <- true
		//webSignalCh <- true
		if viper.GetBool("botsEnabled") {
			botsSignalCh <- true
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
