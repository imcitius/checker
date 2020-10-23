### THIS README IS AUTO-TRANSLATED USING Google Translate.
If you would like to assist me with proper Russian to English translation, please feel free to send PR's or contact me directly. Thanks.


# Implementing checker in Go

The utility is conceived as a universal daemon capable of performing periodic checks (health checks) of various IT systems,
send alerts and perform some actions if the check status changes.

Configuration storage is implemented using the [Koanf] library (https://github.com/knadh/koanf).
By default, the config is loaded from the config.yaml file in the current directory.

``
$ ./checker

Start dev
^ _ ^

Usage:
  checker [command]

Available Commands:
  check Run scheduler and execute checks
  gentoken Generate auth token
  help Help about any command
  list List Projects, Healthchecks, check UUIDs
  testcfg unmarshal config file into config structure
  version Print the version number of Hugo

Flags:
  -b, --bots start listening messenger bots (boolean default true)
  -c, --config string config file
  -f, --configformat string config file format (default "yaml")
  -s, --configsource string config file source: file or consul
  -w, --configwatchtimeout string config watch period (default "5s")
  -D, --debugLevel string Debug level: Debug, Info, Warn, Error, Fatal, Panic (default "info")
  -h, --help help for checker

Use "checker [command] --help" for more information about a command.
``

Configuration storage is available in all repositories supported by the [Koanf] library (https://github.com/knadh/koanf).
You can load any settings from the CHECKER_* environment variables (see the Koanf documentation).

The `-s` switch allows you to switch the receiving of the config to Consul or S3.
For S3, settings are taken from variables:

AWS_ACCESS_KEY_ID - key ID
AWS_SECRET_ACCESS_KEY - secret key
AWS_REGION - region
AWS_BUCKET - bucket name
AWS_OBJECT_KEY - path to the object from the bucket root

For Consul, two ENV variables are read: CONSUL_ADDR and CONSUL_PATH. From the first, the URL of the Consul server is taken, from the second - the path to the KV key with the config.
The KV key must contain the complete configuration, in the formats `yaml`,` json`, `toml`,` hcl` (set by the -f key), loading from the KV tree structure is not supported.

Each period set by the `--configwatchtimeout` key (by default` 5s`) Checker tries to reread the config from the repository. If the config is loaded successfully, its validity and compliance with the current configuration are checked.
If the config is valid and differs from the current configuration, it replaces the current configuration, and the scheduler and bots are restarted.
The config loaded from the file system is also automatically monitored for updates.


Some web endpoints require JWT authorization. JWT token is generated using the CLI command gentoken.
The token is generated using the key specified in the configuration in the `defaults.token_encryption_key` parameter, or in the ENV variable.
ENV variable has higher priority.
Loading a token from a Vault is supported.

The configuration is loaded according to the template (json):
  {
    "defaults": {},
    "actors": {},
    "alerts": {},
    "projects": {},
    "consul_catalog": {}
  }

Secret parameters (passwords, tokens) can be saved in the Hashicorp Vault, at the moment it supports downloading secrets for telegram bots, JWT authorization, passwords for SQL databases and http checks.
Format: `vault: secret / path / to / token: field`. The value of the field `field` from the path` secret / path / to / token` will be used as the secret.
Secrets retrieved from the Vault are cached for 5 minutes to reduce the load on the Vault.

The `defaults` block in the` parameters` subblock describes the default check parameters that are applied to the settings of projects, if they have not been reassigned in the `parameters` block of a specific project.

Separate parameters `timer_step` and` http_port` in the `defaults` block contain the period of checks by the internal timer for the presence of actions that need to be performed at the moment, and the default port for the HTTP server.
If the PORT environment variable is set, the port number from it is used.

## The `parameters` block contains the following settings:

### in defaults and projects
``
run_every: The frequency of testing and running alerts.

// TODO check the feature
min_health: the minimum number of live checks within the healthchck that does not put the project in critical status

allow_fails: the number of checks that have failed to the critical status that can pass before the alert is sent to the critical channel

mode: notification mode, in loud mode alerts are sent to carts, in quiet mode they are only output to stdout.

noncrit_alert: the name of the notification method for non-critical alerts

crit_alert: the name of the alert method for critical alerts

command_channel: the name of the notification method for receiving a command into the bot (by default, the noncritical_channel parameter is taken)

// TODO add certificate checking for all tls, not just https
ssl_expiration_period: checking the proximity of the expiration time of SSL certificates during http checks

// TODO check the feature
periodic_report_time: submission period