package cmd

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"my/checker/alerts"
	"my/checker/config"
	"my/checker/metrics"
	"my/checker/scheduler"
	"my/checker/web"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"sync"
	"syscall"
	"time"
)

var (
	// Used for flags.

	rootCmd = &cobra.Command{
		Use:   "checker",
		Short: "Checker runner",
		Long:  `^_^`,
	}
	signalINT, signalHUP                                                                  chan os.Signal
	configChangeSig, configWatchSig, doneCh, schedulerSignalCh, botsSignalCh, webSignalCh chan bool
	wg                                                                                    sync.WaitGroup
	interrupt                                                                             bool
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "config", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().StringVar(&config.CfgSrc, "configsource", "", "config file source: file or consul (default is file)")
	rootCmd.PersistentFlags().StringVar(&config.CfgWatchTimeout, "configwatchtimeout", "5s", "config watch period (default '5s')")
	rootCmd.PersistentFlags().StringVar(&config.CfgFormat, "configformat", "", "config file format: (default is yaml)")

	rootCmd.PersistentFlags().StringP("debugLevel", "D", "info", "Debug level: Debug,Info,Warn,Error,Fatal,Panic")
	viper.BindPFlag("debugLevel", rootCmd.PersistentFlags().Lookup("debugLevel"))

	rootCmd.PersistentFlags().Bool("bots", true, "start listening messenger bots")
	viper.BindPFlag("botsEnabled", rootCmd.PersistentFlags().Lookup("bots"))

	//rootCmd.PersistentFlags().StringVar(&consulAddr, "consul_addr", "", "Consul server address")
	//rootCmd.PersistentFlags().StringVar(&consulPath, "consul_path", "", "Consul KV path to get config from")
	//rootCmd.PersistentFlags().StringVar(&vaultAddr, "vault_addr", "", "Vault server address")
	//rootCmd.PersistentFlags().StringVar(&vaultToken, "vault_token", "", "Vault token")

	//viper.BindPFlag("vaultToken", rootCmd.PersistentFlags().Lookup("Vault_Token"))
	//viper.BindPFlag("vaultAddr", rootCmd.PersistentFlags().Lookup("Vault_Address"))
	//viper.BindPFlag("consulAddr", rootCmd.PersistentFlags().Lookup("Consul_Address"))
	//viper.BindPFlag("consulPath", rootCmd.PersistentFlags().Lookup("Consul_Path"))

	viper.BindEnv("VAULT_TOKEN")
	viper.BindEnv("VAULT_ADDR")
	viper.BindEnv("CONSUL_ADDR")
	viper.BindEnv("CONSUL_PATH")

	viper.SetDefault("HTTPPort", "80")

	rootCmd.AddCommand(testCfg)
	rootCmd.AddCommand(checkCommand)

	signalINT = make(chan os.Signal)
	signalHUP = make(chan os.Signal)
	doneCh = make(chan bool)
	schedulerSignalCh = make(chan bool)
	webSignalCh = make(chan bool)
	configChangeSig = make(chan bool)
	configWatchSig = make(chan bool)
	botsSignalCh = make(chan bool)
	signal.Notify(signalINT, syscall.SIGINT)
	signal.Notify(signalHUP, syscall.SIGHUP)
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

func initConfig() {

	logrus.Info("initConfig: load config file")
	logrus.Infof("Config flag: %s", config.CfgFile)

	logrus.Infof("%s %s", viper.GetString("CONSUL_ADDR"), viper.GetString("CONSUL_PATH"))

	switch {
	case config.CfgSrc == "" || config.CfgSrc == "file":
		if config.CfgFile == "" {
			// Use config file from the flag.
			viper.SetConfigName("config")         // name of config file (without extension)
			viper.SetConfigType("yaml")           // REQUIRED if the config file does not have the extension in the name
			viper.AddConfigPath("/etc/appname/")  // path to look for the config file in
			viper.AddConfigPath("$HOME/.appname") // call multiple times to add many search paths
			viper.AddConfigPath(".")              // optionally look for config in the working directory

		} else {
			viper.SetConfigName(filepath.Base(config.CfgFile)) // name of config file (without extension)
			if filepath.Ext(config.CfgFile) == "" {
				viper.SetConfigType("yaml") // REQUIRED if the config file does not have the extension in the name
			} else {
				viper.SetConfigType(filepath.Ext(config.CfgFile)[1:])
			}
			viper.AddConfigPath(filepath.Dir(config.CfgFile)) // path to look for the config file in

		}
		viper.WatchConfig()
		viper.OnConfigChange(func(e fsnotify.Event) {
			config.Log.Info("Config file changed: ", e.Name)
			configChangeSig <- true

		})

	case config.CfgSrc == "consul":
		if viper.GetString("CONSUL_ADDR") != "" {
			if viper.GetString("CONSUL_PATH") != "" {
				viper.AddRemoteProvider("consul", viper.GetString("CONSUL_ADDR"), viper.GetString("CONSUL_PATH"))
				viper.SetConfigType("json")
			} else {
				panic("Consul path not specified")
			}
		} else {
			panic("Consul URL not specified")
		}
	}

	viper.AutomaticEnv()

	dl, err := logrus.ParseLevel(viper.GetString("debugLevel"))
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

func mainChecker() {
	go watchConfig()

	for {
		config.Log.Info("Start main loop")
		interrupt = false

		err := config.LoadConfig()
		if err != nil {
			config.Log.Infof("Config load error: %s", err)
		} else {
			config.Log.Debugf("Loaded config: %+v", config.Config)
		}

		err = metrics.InitMetrics()
		if err != nil {
			config.Log.Infof("Metrics init error: %s", err)
		}

		go signalWait()

		if config.Sem.TryAcquire(1) {
			config.Log.Debugf("Fire webserver")
			go web.WebInterface(webSignalCh, config.Sem)
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
		webSignalCh <- true
	case <-signalHUP:
		config.Log.Infof("Got SIGHUP")
		configChangeSig <- true
	case <-configChangeSig:
		config.Log.Infof("Config file reload")
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
		_, err := config.TestConfig()
		if err != nil {
			config.Log.Infof("Config loading error: %+v", err)
		} else {
			config.Log.Infof("Config load ok (err: %+v)", err)
		}
	},
}

func watchConfig() {
	for {
		if period, err := time.ParseDuration(config.CfgWatchTimeout); err != nil {
			config.Log.Infof("KV watch timeout parser error: %+v, use 5s", err)
			time.Sleep(time.Second * 5) // default delay
		} else {
			time.Sleep(period)
		}
		tempConfig, err := config.TestConfig()
		if err == nil {
			if reflect.DeepEqual(config.Config, tempConfig) {
				config.Log.Infof("KV config not changed: %s", err)
			} else {
				config.Log.Infof("KV config changed, reloading")
				err := config.LoadConfig()
				if err != nil {
					config.Log.Infof("Config load error: %s", err)
				} else {
					config.Log.Debugf("Loaded config: %+v", config.Config)
				}
				configChangeSig <- true
			}
		} else {
			config.Log.Infof("KV config seems to be broken: %+v", err)
		}

		//configWatchSig <- true
	}
}
