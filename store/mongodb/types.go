package store

type DBConfig struct {
	Protocol string `yaml:"protocol" env-default:""`
	Host     string `yaml:"host" env-default:""`
	Port     string `yaml:"port" env-default:""`
	Username string `yaml:"username" env-default:""`
	Password string `yaml:"password" env-default:""`
}
