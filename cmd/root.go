package cmd

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"my/checker/alerts"
	"my/checker/config"
	"my/checker/scheduler"
	"my/checker/status"
	"my/checker/web"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var (
	// Used for flags.

	rootCmd = &cobra.Command{
		Use:   "checker",
		Short: "Checker runner",
		Long:  `^_^`,
	}
	interrupt bool
)

// Execute executes the root command.
func Execute() error {
	if config.Version != "" && config.VersionSHA != "" && config.VersionBuild != "" {
		fmt.Printf("Start %s (commit: %s; build: %s)\n", config.Version, config.VersionSHA, config.VersionBuild)
	} else {
		fmt.Println("Start dev ")
	}

	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(config.InitConfig)

	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "config", "config file")
	rootCmd.PersistentFlags().StringVar(&config.CfgSrc, "configsource", "file", "config file source: file or consul")
	rootCmd.PersistentFlags().StringVar(&config.CfgWatchTimeout, "configwatchtimeout", "5s", "config watch period")
	rootCmd.PersistentFlags().StringVar(&config.CfgFormat, "configformat", "yaml", "config file format")

	rootCmd.PersistentFlags().StringP("debugLevel", "D", "info", "Debug level: Debug,Info,Warn,Error,Fatal,Panic")
	config.Viper.BindPFlag("debugLevel", rootCmd.PersistentFlags().Lookup("debugLevel"))

	rootCmd.PersistentFlags().Bool("bots", true, "start listening messenger bots")
	config.Viper.BindPFlag("botsEnabled", rootCmd.PersistentFlags().Lookup("bots"))

	config.Viper.BindEnv("VAULT_TOKEN")
	config.Viper.BindEnv("VAULT_ADDR")
	config.Viper.BindEnv("CONSUL_ADDR")
	config.Viper.BindEnv("CONSUL_PATH")
	config.Viper.BindEnv("PORT")

	config.Viper.SetDefault("HTTPPort", "80")
	if config.Viper.GetString("PORT") != "" {
		config.Viper.SetDefault("HTTPPort", config.Viper.GetString("PORT"))
	}

	rootCmd.AddCommand(testCfg)
	rootCmd.AddCommand(checkCommand)

	config.SignalINT = make(chan os.Signal)
	config.SignalHUP = make(chan os.Signal)
	//config.DoneCh = make(chan bool)
	config.SchedulerSignalCh = make(chan bool)
	config.WebSignalCh = make(chan bool)
	config.ConfigChangeSig = make(chan bool)
	//config.ConfigWatchSig = make(chan bool)
	config.BotsSignalCh = make(chan bool)
	signal.Notify(config.SignalINT, syscall.SIGINT)
	signal.Notify(config.SignalHUP, syscall.SIGHUP)
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

func initConfig() {

	logrus.Info("initConfig: load config file")
	logrus.Infof("Config flag: %s", config.CfgFile)

	logrus.Debugf("%s %s", config.Viper.GetString("CONSUL_ADDR"), config.Viper.GetString("CONSUL_PATH"))

	switch {
	case config.CfgSrc == "" || config.CfgSrc == "file":
		if config.CfgFile == "" {
			// Use config file from the flag.
			config.Viper.SetConfigName("config")         // name of config file (without extension)
			config.Viper.SetConfigType("yaml")           // REQUIRED if the config file does not have the extension in the name
			config.Viper.AddConfigPath("/etc/appname/")  // path to look for the config file in
			config.Viper.AddConfigPath("$HOME/.appname") // call multiple times to add many search paths
			config.Viper.AddConfigPath(".")              // optionally look for config in the working directory

		} else {
			config.Viper.SetConfigName(filepath.Base(config.CfgFile)) // name of config file (without extension)
			if filepath.Ext(config.CfgFile) == "" {
				config.Viper.SetConfigType("yaml") // REQUIRED if the config file does not have the extension in the name
			} else {
				config.Viper.SetConfigType(filepath.Ext(config.CfgFile)[1:])
			}
			config.Viper.AddConfigPath(filepath.Dir(config.CfgFile)) // path to look for the config file in

		}
		config.Viper.WatchConfig()
		config.Viper.OnConfigChange(func(e fsnotify.Event) {
			config.Log.Info("Config file changed: ", e.Name)
			config.ConfigChangeSig <- true

		})

	case config.CfgSrc == "consul":
		if config.Viper.GetString("CONSUL_ADDR") != "" {
			if config.Viper.GetString("CONSUL_PATH") != "" {
				config.Viper.AddRemoteProvider("consul", config.Viper.GetString("CONSUL_ADDR"), config.Viper.GetString("CONSUL_PATH"))
				config.Viper.SetConfigType("json")
			} else {
				panic("Consul path not specified")
			}
		} else {
			panic("Consul URL not specified")
		}
	}

	config.Viper.AutomaticEnv()

	dl, err := logrus.ParseLevel(config.Viper.GetString("debugLevel"))
	if err != nil {
		config.Log.Panicf("Cannot parse debug level: %v", err)
	} else {
		config.Log.SetLevel(dl)
	}

}

var checkCommand = &cobra.Command{
	Use:   "check",
	Short: "Run scheduler and execute checks",
	Run: func(cmd *cobra.Command, args []string) {
		mainChecker()
	},
}

var testCfg = &cobra.Command{
	Use:   "testcfg",
	Short: "unmarshal config file into config structure",
	Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		testConfig()
	},
}

func mainChecker() {

	for {
		config.Log.Info("Start main loop")
		go signalWait()
		interrupt = false

		err := config.LoadConfig()
		if err != nil {
			config.Log.Infof("Config load error: %s", err)
		}
		//else {
		//	config.Log.Debugf("(mainChecker) Loaded config: %+v", config.Config)
		//}

		err = status.InitStatuses()
		if err != nil {
			config.Log.Infof("Status init error: %s", err)
		}

		if config.Sem.TryAcquire(1) {
			config.Log.Debugf("Fire webserver")
			go web.WebInterface(config.WebSignalCh, config.Sem)
		} else {
			config.Log.Debugf("Webserver already running")
		}

		config.Wg.Add(1)
		config.Log.Debugf("Fire scheduler")
		go scheduler.RunScheduler(config.SchedulerSignalCh, &config.Wg)

		config.Log.Debugf("botsEnabled is %v", config.Viper.GetBool("botsEnabled"))
		if config.Viper.GetBool("botsEnabled") {
			config.Log.Debugf("Fire bots")
			config.Wg.Add(1)
			commandChannel, err := alerts.GetCommandChannel()
			if err != nil {
				config.Log.Infof("root GetCommandChannel error: %s", err)
			} else {
				a := alerts.GetAlertProto(commandChannel)
				if a == nil {
					config.Log.Fatal("root commandChannel not found, bot not init")
				} else {
					a.InitBot(config.BotsSignalCh, &config.Wg)
				}
			}
		}

		config.InternalStatus = "started"

		config.Wg.Wait()

		if !interrupt {
			config.Log.Debug("Checker init complete")
		} else {
			config.Log.Debug("Checker stopped")
			os.Exit(1)
		}
	}
}

func testConfig() {
	_, err := config.TestConfig()
	if err != nil {
		config.Log.Infof("Config loading error: %+v", err)
	} else {
		config.Log.Infof("Config load ok (err: %+v)", err)
	}
}

func signalWait() {
	config.Log.Info("Start waiting signals")
	select {
	case <-config.SignalINT:
		config.Log.Infof("Got SIGINT")
		config.InternalStatus = "stop"
		interrupt = true
		config.SchedulerSignalCh <- true
		if config.Viper.GetBool("botsEnabled") {
			config.BotsSignalCh <- true
		}
		config.WebSignalCh <- true
		return
	case <-config.SignalHUP:
		config.Log.Infof("Got SIGHUP")
		config.ConfigChangeSig <- true
		return
	case <-config.ConfigChangeSig:
		config.Log.Infof("Config file reload")
		config.InternalStatus = "reload"
		config.SchedulerSignalCh <- true
		//config.WebSignalCh <- true
		if config.Viper.GetBool("botsEnabled") {
			config.BotsSignalCh <- true
		}
		return
	}
}
