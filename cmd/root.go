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

	debugLevel, configFile, configSource, configWatchTimeout, configFormat string
	botsEnabled                                                            bool
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
	rootCmd.PersistentFlags().StringVarP(&debugLevel, "debugLevel", "D", "info", "Debug level: Debug,Info,Warn,Error,Fatal,Panic")
	rootCmd.PersistentFlags().BoolVarP(&botsEnabled, "bots", "b", true, "start listening messenger bots")

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

	config.Koanf.Load(confmap.Provider(map[string]interface{}{
		"defaults.http.port":    "80",
		"defaults.http.enabled": true,
		"debug.level":           debugLevel,
		"bots.enabled":          botsEnabled,
		"config.file":           configFile,
		"config.source":         configSource,
		"config.watchtimeout":   configWatchTimeout,
		"config.format":         configFormat,
	}, "."), nil)

	config.Koanf.Load(env.Provider("PORT", ".", func(s string) string { return "defaults.http.port" }), nil)
	config.Koanf.Load(env.Provider("CONSUL_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			s), "_", ".", -1)
	}), nil)
	config.Koanf.Load(env.Provider("VAULT_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			s), "_", ".", -1)
	}), nil)
	config.Koanf.Load(env.Provider("AWS_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			s), "_", ".", -1)
	}), nil)
	config.Koanf.Load(env.Provider("CHECKER_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			s), "_", ".", -1)
	}), nil)

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

		config.Log.Debugf("botsEnabled is %v", config.Koanf.Bool("bots.enabled"))
		if config.Koanf.Bool("bots.enabled") {
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
		if config.Koanf.Bool("bots.enabled") {
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
		if config.Koanf.Bool("bots.enabled") {
			config.BotsSignalCh <- true
		}
		return
	}
}
