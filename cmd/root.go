package cmd

import (
	"fmt"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"my/checker/alerts"
	"my/checker/auth"
	"my/checker/catalog"
	"my/checker/config"
	"my/checker/reports"
	"my/checker/scheduler"
	"my/checker/status"
	"my/checker/web"
	"os"
	"os/signal"
	"strings"
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

	logFormat, debugLevel, configFile, configSource, configWatchTimeout, configFormat string
	botsEnabled                                                                       bool
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

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file")
	rootCmd.PersistentFlags().StringVarP(&configSource, "configsource", "s", "", "config file source: file, consul, s3")
	rootCmd.PersistentFlags().StringVarP(&configWatchTimeout, "configwatchtimeout", "w", "5s", "config watch period")
	rootCmd.PersistentFlags().StringVarP(&configFormat, "configformat", "f", "yaml", "config file format")
	rootCmd.PersistentFlags().StringVarP(&logFormat, "logformat", "l", "text", "log format: text/json")
	rootCmd.PersistentFlags().StringVarP(&debugLevel, "debugLevel", "D", "warn", "Debug level: Debug,Info,Warn,Error,Fatal,Panic")
	rootCmd.PersistentFlags().BoolVarP(&config.BotsEnabledFlag, "botsenabled", "b", false, "Whether to enable active bots")

	rootCmd.AddCommand(genToken)
	rootCmd.AddCommand(testCfg)
	rootCmd.AddCommand(checkCommand)
	rootCmd.AddCommand(list)

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

	logrus.Info("initConfig: load config file")
	//logrus.Infof("Config file: %s", config.Koanf.String("config.file"))
	//logrus.Infof("Config type: %s", config.Koanf.String("config.source"))

}

func initConfig() {

	err := config.Koanf.Load(confmap.Provider(map[string]interface{}{
		"defaults.http.port":    "80",
		"defaults.http.enabled": true,
		"debug.level":           debugLevel,
		"log.format":            logFormat,
		"bots.enabled":          botsEnabled,
		"config.file":           configFile,
		"config.source":         configSource,
		"config.watchtimeout":   configWatchTimeout,
		"config.format":         configFormat,
	}, "."), nil)
	if err != nil {
		logrus.Panicf("Cannot fill default config: %s", err.Error())
	}

	err = config.Koanf.Load(env.Provider("PORT", ".", func(s string) string {
		return "defaults.http.port"
	}), nil)
	if err != nil {
		logrus.Infof("PORT env not defined: %s", err.Error())
	}

	err = config.Koanf.Load(env.Provider("DEBUG_LEVEL", ".", func(s string) string {
		return "debug.level"
	}), nil)
	if err != nil {
		logrus.Infof("DEBUG_LEVEL env not defined: %s", err.Error())
	}

	for _, i := range []string{"CONSUL_", "VAULT_", "AWS_", "CHECKER_"} {
		err = config.Koanf.Load(env.Provider(i, ".", func(s string) string {
			return strings.Replace(strings.ToLower(
				s), "_", ".", -1)
		}), nil)
		if err != nil {
			logrus.Infof("%s env not defined: %s", i, err.Error())
		}
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
	Long:  `Try to load and parse config from defined source`,
	Run: func(cmd *cobra.Command, args []string) {
		testConfig()
	},
}

var genToken = &cobra.Command{
	Use:   "gentoken",
	Short: "Generate auth token",
	Long:  `Generate new jwt token for web auth`,
	Run: func(cmd *cobra.Command, args []string) {
		auth.GenerateToken()
	},
}

var list = &cobra.Command{
	Use:   "list",
	Short: "List config elements",
	Long:  `List Projects, Healthchecks, Check UUIDs`,
	Run: func(cmd *cobra.Command, args []string) {
		err := config.LoadConfig()
		if err != nil {
			config.Log.Infof("Config load error: %s", err)
		}

		if config.Config.ConsulCatalog.Enabled {
			cat, err := catalog.GetConsulServices()
			if err != nil {
				config.Log.Errorf("Failed to get consul services: %s", err)
				//notifyError(err)
				return
			}
			catalog.ParseCatalog(cat)
		}
		reports.List()
	},
}

func fireActiveBot() {
	config.Log.Infof("Active bot is enabled")
	fireBot()

}
func firePassiveBot() {
	config.Log.Infof("Active bot is disabled, alerts only")
	message := fmt.Sprintf("Bot at your service (%s, %s, %s)\nActive bot is disabled, alerts only", config.Version, config.VersionSHA, config.VersionBuild)
	// Metrics structures is not initialized yet, so we prevent panic with "noMetrics"
	alerts.SendChatOps(message, "noMetrics")
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

		if config.Config.ConsulCatalog.Enabled {
			catalog.WatchServices()
		}

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

		config.Log.Debugf("config botsEnabled is %v", config.Config.Defaults.BotsEnabled)

		switch config.Config.Defaults.BotsEnabled {
		case true:
			fireActiveBot()
			//if config.BotsEnabledFlag {
			//	fireActiveBot()
			//} else {
			//	firePassiveBot()
			//}
		case false:
			firePassiveBot()
			//if config.BotsEnabledFlag {
			//	fireActiveBot()
			//} else {
			//	firePassiveBot()
			//}
		}

		//config.InternalStatus = "started"

		config.Wg.Wait()

		if !interrupt {
			config.Log.Debug("Checker init complete")
		} else {
			config.Log.Debug("Checker stopped")
			os.Exit(1)
		}
	}
}

func fireBot() {
	config.Log.Debugf("Fire bot")
	config.Wg.Add(1)
	commandChannel, err := alerts.GetCommandChannel()
	if err != nil {
		config.Log.Infof("root GetCommandChannel error: %s", err)
	} else {
		a := commandChannel.GetAlertProto()
		if a == nil {
			config.Log.Fatal("root commandChannel not found, bot not initialized")
		} else {
			a.InitBot(config.BotsSignalCh, &config.Wg)
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
		if config.Config.Defaults.BotsEnabled {
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
		if config.Config.Defaults.BotsEnabled {
			config.BotsSignalCh <- true
		}
		return
	}
}
