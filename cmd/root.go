package cmd

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"my/checker/auth"
	"my/checker/catalog"
	"my/checker/config"
	"my/checker/reports"
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

	logFormat, debugLevel, configFile, configSource, configWatchTimeout, configFormat string
	checkUUID                                                                         string
	botsEnabled, watchConfig                                                          bool
)

// Execute executes the root command.
func Execute() error {
	fmt.Printf("Start %s (commit: %s; build: %s)\n", config.Version, config.VersionSHA, config.VersionBuild)
	return rootCmd.Execute()
}

func init() {

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file")
	rootCmd.PersistentFlags().StringVarP(&configSource, "configsource", "s", "", "config file source: file, consul, s3, env")
	rootCmd.PersistentFlags().StringVarP(&configFormat, "configformat", "f", "yaml", "config file format")
	rootCmd.PersistentFlags().StringVarP(&configWatchTimeout, "configwatchtimeout", "w", "5s", "config watch period")
	rootCmd.PersistentFlags().BoolVarP(&watchConfig, "watchConfig", "W", false, "Whether to watch config file changes on disk")
	rootCmd.PersistentFlags().StringVarP(&logFormat, "logformat", "l", "text", "log format: text/json")
	rootCmd.PersistentFlags().StringVarP(&debugLevel, "debugLevel", "D", "warn", "Debug level: Debug,Info,Warn,Error,Fatal,Panic")
	rootCmd.PersistentFlags().StringVarP(&checkUUID, "checkUUID", "u", "", "UUID to check with SingleCheck")
	rootCmd.PersistentFlags().BoolVarP(&botsEnabled, "botsEnabled", "b", false, "Whether to enable active bot")

	rootCmd.AddCommand(genToken)
	rootCmd.AddCommand(testCfg)
	rootCmd.AddCommand(checkCommand)
	rootCmd.AddCommand(singleCheckCommand)
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

var checkCommand = &cobra.Command{
	Use:   "check",
	Short: "run scheduler and execute checks",
	Run: func(cmd *cobra.Command, args []string) {
		mainChecker()
	},
}

var singleCheckCommand = &cobra.Command{
	Use:   "singlecheck",
	Short: "execute single check by UUID",
	Run: func(cmd *cobra.Command, args []string) {
		singleCheck()
	},
}

var testCfg = &cobra.Command{
	Use:   "testcfg",
	Short: "unmarshal config file into config structure",
	Long:  `try to load and parse config from defined source`,
	Run: func(cmd *cobra.Command, args []string) {
		testConfig()
	},
}

var genToken = &cobra.Command{
	Use:   "gentoken",
	Short: "generate auth token",
	Long:  `generate new jwt token for web auth`,
	Run: func(cmd *cobra.Command, args []string) {
		auth.GenerateToken()
	},
}

var list = &cobra.Command{
	Use:   "list",
	Short: "list config elements",
	Long:  `list Projects, Healthchecks, Check UUIDs`,
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
		fmt.Println(reports.List())
	},
}
