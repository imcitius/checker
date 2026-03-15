package models

import "strconv"

// CheckConfig is the interface that all check configurations must implement
type CheckConfig interface {
	CheckType() string
	GetTarget() string
}

// Ensure structs implement the interface
var _ CheckConfig = (*HTTPCheckConfig)(nil)
var _ CheckConfig = (*TCPCheckConfig)(nil)
var _ CheckConfig = (*ICMPCheckConfig)(nil)
var _ CheckConfig = (*PassiveCheckConfig)(nil)
var _ CheckConfig = (*MySQLCheckConfig)(nil)
var _ CheckConfig = (*PostgreSQLCheckConfig)(nil)
var _ CheckConfig = (*SSHCheckConfig)(nil)
var _ CheckConfig = (*RedisCheckConfig)(nil)
var _ CheckConfig = (*MongoDBCheckConfig)(nil)
var _ CheckConfig = (*WebhookConfig)(nil)
var _ CheckConfig = (*DomainExpiryCheckConfig)(nil)

// AuthConfig holds authentication credentials
type AuthConfig struct {
	User     string `bson:"user,omitempty" json:"user,omitempty"`
	Password string `bson:"password,omitempty" json:"password,omitempty"`
}

// HTTPCheckConfig holds configuration for HTTP checks
type HTTPCheckConfig struct {
	URL                 string              `bson:"url,omitempty" json:"url,omitempty"`
	Timeout             string              `bson:"timeout,omitempty" json:"timeout,omitempty"`
	Answer              string              `bson:"answer,omitempty" json:"answer,omitempty"`
	AnswerPresent       bool                `bson:"answer_present,omitempty" json:"answer_present,omitempty"`
	Code                []int               `bson:"code,omitempty" json:"code,omitempty"`
	Headers             []map[string]string `bson:"headers,omitempty" json:"headers,omitempty"`
	Cookies             []map[string]string `bson:"cookies,omitempty" json:"cookies,omitempty"`
	SkipCheckSSL        bool                `bson:"skip_check_ssl,omitempty" json:"skip_check_ssl,omitempty"`
	SSLExpirationPeriod string              `bson:"ssl_expiration_period,omitempty" json:"ssl_expiration_period,omitempty"`
	StopFollowRedirects bool                `bson:"stop_follow_redirects,omitempty" json:"stop_follow_redirects,omitempty"`
	Auth                AuthConfig          `bson:"auth,omitempty" json:"auth,omitempty"`
}

func (c *HTTPCheckConfig) CheckType() string { return "http" }
func (c *HTTPCheckConfig) GetTarget() string { return c.URL }

// TCPCheckConfig holds configuration for TCP checks
type TCPCheckConfig struct {
	Host    string `bson:"host,omitempty" json:"host,omitempty"`
	Port    int    `bson:"port,omitempty" json:"port,omitempty"`
	Timeout string `bson:"timeout,omitempty" json:"timeout,omitempty"`
}

func (c *TCPCheckConfig) CheckType() string { return "tcp" }
func (c *TCPCheckConfig) GetTarget() string { return c.Host + ":" + strconv.Itoa(c.Port) }

// ICMPCheckConfig holds configuration for ICMP (Ping) checks
type ICMPCheckConfig struct {
	Host    string `bson:"host,omitempty" json:"host,omitempty"`
	Count   int    `bson:"count,omitempty" json:"count,omitempty"`
	Timeout string `bson:"timeout,omitempty" json:"timeout,omitempty"`
}

func (c *ICMPCheckConfig) CheckType() string { return "icmp" }
func (c *ICMPCheckConfig) GetTarget() string { return c.Host }

// PassiveCheckConfig is a placeholder (might need more fields later)
type PassiveCheckConfig struct {
	Timeout string `bson:"timeout,omitempty" json:"timeout,omitempty"`
}

func (c *PassiveCheckConfig) CheckType() string { return "passive" }
func (c *PassiveCheckConfig) GetTarget() string { return "" }

// MySQLCheckConfig holds configuration for MySQL checks
type MySQLCheckConfig struct {
	Host       string   `bson:"host,omitempty" json:"host,omitempty"`
	Port       int      `bson:"port,omitempty" json:"port,omitempty"`
	Timeout    string   `bson:"timeout,omitempty" json:"timeout,omitempty"`
	UserName   string   `bson:"username,omitempty" json:"username,omitempty"`
	Password   string   `bson:"password,omitempty" json:"password,omitempty"`
	DBName     string   `bson:"dbname,omitempty" json:"dbname,omitempty"`
	Query      string   `bson:"query,omitempty" json:"query,omitempty"`
	Response   string   `bson:"response,omitempty" json:"response,omitempty"`
	Difference string   `bson:"difference,omitempty" json:"difference,omitempty"`
	TableName  string   `bson:"table_name,omitempty" json:"table_name,omitempty"`
	Lag        string   `bson:"lag,omitempty" json:"lag,omitempty"`
	ServerList []string `bson:"server_list,omitempty" json:"server_list,omitempty"`
}

func (c *MySQLCheckConfig) CheckType() string { return "mysql" }
func (c *MySQLCheckConfig) GetTarget() string { return c.Host + ":" + strconv.Itoa(c.Port) }

// PostgreSQLCheckConfig holds configuration for PostgreSQL checks
type PostgreSQLCheckConfig struct {
	Host             string   `bson:"host,omitempty" json:"host,omitempty"`
	Port             int      `bson:"port,omitempty" json:"port,omitempty"`
	Timeout          string   `bson:"timeout,omitempty" json:"timeout,omitempty"`
	UserName         string   `bson:"username,omitempty" json:"username,omitempty"`
	Password         string   `bson:"password,omitempty" json:"password,omitempty"`
	DBName           string   `bson:"dbname,omitempty" json:"dbname,omitempty"`
	SSLMode          string   `bson:"sslmode,omitempty" json:"sslmode,omitempty"`
	Query            string   `bson:"query,omitempty" json:"query,omitempty"`
	Response         string   `bson:"response,omitempty" json:"response,omitempty"`
	Difference       string   `bson:"difference,omitempty" json:"difference,omitempty"`
	TableName        string   `bson:"table_name,omitempty" json:"table_name,omitempty"`
	Lag              string   `bson:"lag,omitempty" json:"lag,omitempty"`
	ServerList       []string `bson:"server_list,omitempty" json:"server_list,omitempty"`
	AnalyticReplicas []string `bson:"analytic_replicas,omitempty" json:"analytic_replicas,omitempty"`
}

func (c *PostgreSQLCheckConfig) CheckType() string { return "pgsql" }
func (c *PostgreSQLCheckConfig) GetTarget() string { return c.Host + ":" + strconv.Itoa(c.Port) }

// SSHCheckConfig holds configuration for SSH banner checks
type SSHCheckConfig struct {
	Host         string `bson:"host,omitempty" json:"host,omitempty"`
	Port         int    `bson:"port,omitempty" json:"port,omitempty"`
	Timeout      string `bson:"timeout,omitempty" json:"timeout,omitempty"`
	ExpectBanner string `bson:"expect_banner,omitempty" json:"expect_banner,omitempty"`
}

func (c *SSHCheckConfig) CheckType() string { return "ssh" }
func (c *SSHCheckConfig) GetTarget() string { return c.Host + ":" + strconv.Itoa(c.Port) }

// RedisCheckConfig holds configuration for Redis checks
type RedisCheckConfig struct {
	Host     string `bson:"host,omitempty" json:"host,omitempty"`
	Port     int    `bson:"port,omitempty" json:"port,omitempty"`
	Timeout  string `bson:"timeout,omitempty" json:"timeout,omitempty"`
	Password string `bson:"password,omitempty" json:"password,omitempty"`
	DB       int    `bson:"db,omitempty" json:"db,omitempty"`
}

func (c *RedisCheckConfig) CheckType() string { return "redis" }
func (c *RedisCheckConfig) GetTarget() string { return c.Host + ":" + strconv.Itoa(c.Port) }

// MongoDBCheckConfig holds configuration for MongoDB checks
type MongoDBCheckConfig struct {
	URI     string `bson:"uri,omitempty" json:"uri,omitempty"`
	Timeout string `bson:"timeout,omitempty" json:"timeout,omitempty"`
}

func (c *MongoDBCheckConfig) CheckType() string { return "mongodb" }
func (c *MongoDBCheckConfig) GetTarget() string { return c.URI }

// DomainExpiryCheckConfig holds configuration for domain expiry WHOIS checks
type DomainExpiryCheckConfig struct {
	Domain            string `bson:"domain,omitempty" json:"domain,omitempty"`
	Timeout           string `bson:"timeout,omitempty" json:"timeout,omitempty"`
	ExpiryWarningDays int    `bson:"expiry_warning_days,omitempty" json:"expiry_warning_days,omitempty"`
}

func (c *DomainExpiryCheckConfig) CheckType() string { return "domain_expiry" }
func (c *DomainExpiryCheckConfig) GetTarget() string { return c.Domain }

// WebhookConfig is for the Webhook Actor (but was mixed in CheckDefinition)
type WebhookConfig struct {
	URL     string            `bson:"url,omitempty" json:"url,omitempty"`
	Method  string            `bson:"method,omitempty" json:"method,omitempty"`
	Payload string            `bson:"payload,omitempty" json:"payload,omitempty"`
	Headers map[string]string `bson:"headers,omitempty" json:"headers,omitempty"`
}

func (c *WebhookConfig) CheckType() string { return "webhook" }
func (c *WebhookConfig) GetTarget() string { return c.URL }
