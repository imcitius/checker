package alerts

type TAlertsConfig struct {
	Alerts *map[string]TAlert
}

type TAlert struct {
	Name            string `yaml:"name"`
	Type            string `yaml:"type"`
	BotToken        string `yaml:"bot_token"`
	ProjectChannel  string `yaml:"noncritical_channel"`
	CriticalChannel string `yaml:"critical_channel"`
}

type TAlertDetails struct {
	Severity    string
	Message     string
	UUID        string
	ProjectName string
}
