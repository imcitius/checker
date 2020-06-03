package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"my/checker/alerts"
	"my/checker/config"
	"my/checker/metrics"
	"my/checker/scheduler"
	"my/checker/web"
	"os"
	"os/signal"
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

	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "config", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().StringVar(&config.CfgSrc, "configsource", "", "config file source: file or consul (default is file)")
	rootCmd.PersistentFlags().StringVar(&config.CfgWatchTimeout, "configwatchtimeout", "5s", "config watch period (default '5s')")
	rootCmd.PersistentFlags().StringVar(&config.CfgFormat, "configformat", "yaml", "config file format: (default is yaml)")

	rootCmd.PersistentFlags().StringP("debugLevel", "D", "info", "Debug level: Debug,Info,Warn,Error,Fatal,Panic")
	config.Viper.BindPFlag("debugLevel", rootCmd.PersistentFlags().Lookup("debugLevel"))

	rootCmd.PersistentFlags().Bool("bots", true, "start listening messenger bots")
	config.Viper.BindPFlag("botsEnabled", rootCmd.PersistentFlags().Lookup("bots"))

	//rootCmd.PersistentFlags().StringVar(&consulAddr, "consul_addr", "", "Consul server address")
	//rootCmd.PersistentFlags().StringVar(&consulPath, "consul_path", "", "Consul KV path to get config from")
	//rootCmd.PersistentFlags().StringVar(&vaultAddr, "vault_addr", "", "Vault server address")
	//rootCmd.PersistentFlags().StringVar(&vaultToken, "vault_token", "", "Vault token")

	//config.Viper.BindPFlag("vaultToken", rootCmd.PersistentFlags().Lookup("Vault_Token"))
	//config.Viper.BindPFlag("vaultAddr", rootCmd.PersistentFlags().Lookup("Vault_Address"))
	//config.Viper.BindPFlag("consulAddr", rootCmd.PersistentFlags().Lookup("Consul_Address"))
	//config.Viper.BindPFlag("consulPath", rootCmd.PersistentFlags().Lookup("Consul_Path"))

	config.Viper.BindEnv("VAULT_TOKEN")
	config.Viper.BindEnv("VAULT_ADDR")
	config.Viper.BindEnv("CONSUL_ADDR")
	config.Viper.BindEnv("CONSUL_PATH")

	config.Viper.SetDefault("HTTPPort", "80")

	rootCmd.AddCommand(testCfg)
	rootCmd.AddCommand(checkCommand)

	config.SignalINT = make(chan os.Signal)
	config.SignalHUP = make(chan os.Signal)
	config.DoneCh = make(chan bool)
	config.SchedulerSignalCh = make(chan bool)
	config.WebSignalCh = make(chan bool)
	config.ConfigChangeSig = make(chan bool)
	config.ConfigWatchSig = make(chan bool)
	config.BotsSignalCh = make(chan bool)
	signal.Notify(config.SignalINT, syscall.SIGINT)
	signal.Notify(config.SignalHUP, syscall.SIGHUP)
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
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
	go config.WatchConfig()

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
			go web.WebInterface(config.WebSignalCh, config.Sem)
		} else {
			config.Log.Debugf("Webserver already running")
		}

		config.Wg.Add(1)
		go scheduler.RunScheduler(config.SchedulerSignalCh, &config.Wg)

		if config.Viper.GetBool("botsEnabled") {
			config.Log.Debugf("botsEnabled is %v", config.Viper.GetBool("botsEnabled"))
			config.Wg.Add(1)
			alerts.InitBots(config.BotsSignalCh, &config.Wg)
		}

		config.Wg.Wait()

		if interrupt {
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
	select {
	case <-config.SignalINT:
		config.Log.Infof("Got SIGINT")
		interrupt = true
		if config.Viper.GetBool("botsEnabled") {
			config.BotsSignalCh <- true
		}
		config.SchedulerSignalCh <- true
		config.WebSignalCh <- true
	case <-config.SignalHUP:
		config.Log.Infof("Got SIGHUP")
		config.ConfigChangeSig <- true
	case <-config.ConfigChangeSig:
		config.Log.Infof("Config file reload")
		config.SchedulerSignalCh <- true
		//webSignalCh <- true
		if config.Viper.GetBool("botsEnabled") {
			config.BotsSignalCh <- true
		}
	}
}
