package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckConfig_InterfaceCompliance(t *testing.T) {
	configs := []CheckConfig{
		&HTTPCheckConfig{URL: "https://example.com"},
		&TCPCheckConfig{Host: "example.com", Port: 80},
		&ICMPCheckConfig{Host: "example.com"},
		&PassiveCheckConfig{Timeout: "30s"},
		&MySQLCheckConfig{Host: "db.example.com", Port: 3306},
		&PostgreSQLCheckConfig{Host: "db.example.com", Port: 5432},
		&DNSCheckConfig{Domain: "example.com"},
		&SSHCheckConfig{Host: "ssh.example.com", Port: 22},
		&RedisCheckConfig{Host: "redis.example.com", Port: 6379},
		&MongoDBCheckConfig{URI: "mongodb://localhost:27017"},
		&DomainExpiryCheckConfig{Domain: "example.com"},
		&WebhookConfig{URL: "https://hook.example.com"},
		&SSLCertCheckConfig{Host: "example.com", Port: 443},
		&SMTPCheckConfig{Host: "smtp.example.com", Port: 25},
		&GRPCHealthCheckConfig{Host: "grpc.example.com:50051"},
		&WebSocketCheckConfig{URL: "wss://ws.example.com"},
	}

	for _, c := range configs {
		assert.NotEmpty(t, c.CheckType(), "CheckType() should not be empty for %T", c)
	}
}

func TestHTTPCheckConfig_CheckType(t *testing.T) {
	c := &HTTPCheckConfig{}
	assert.Equal(t, "http", c.CheckType())
}

func TestHTTPCheckConfig_GetTarget(t *testing.T) {
	c := &HTTPCheckConfig{URL: "https://api.example.com/health"}
	assert.Equal(t, "https://api.example.com/health", c.GetTarget())
}

func TestHTTPCheckConfig_GetTarget_Empty(t *testing.T) {
	c := &HTTPCheckConfig{}
	assert.Equal(t, "", c.GetTarget())
}

func TestHTTPCheckConfig_JSONRoundTrip(t *testing.T) {
	c := HTTPCheckConfig{
		URL:                 "https://example.com",
		Timeout:             "10s",
		Answer:              "ok",
		AnswerPresent:       true,
		Code:                []int{200, 201},
		SkipCheckSSL:        true,
		StopFollowRedirects: true,
		Auth:                AuthConfig{User: "admin", Password: "secret"},
	}

	data, err := json.Marshal(c)
	assert.NoError(t, err)

	var got HTTPCheckConfig
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, c.URL, got.URL)
	assert.Equal(t, c.Code, got.Code)
	assert.Equal(t, c.Auth.User, got.Auth.User)
	assert.True(t, got.SkipCheckSSL)
	assert.True(t, got.StopFollowRedirects)
}

func TestTCPCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &TCPCheckConfig{Host: "example.com", Port: 8080}
	assert.Equal(t, "tcp", c.CheckType())
	assert.Equal(t, "example.com:8080", c.GetTarget())
}

func TestTCPCheckConfig_ZeroPort(t *testing.T) {
	c := &TCPCheckConfig{Host: "example.com", Port: 0}
	assert.Equal(t, "example.com:0", c.GetTarget())
}

func TestICMPCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &ICMPCheckConfig{Host: "8.8.8.8", Count: 3, Timeout: "5s"}
	assert.Equal(t, "icmp", c.CheckType())
	assert.Equal(t, "8.8.8.8", c.GetTarget())
}

func TestPassiveCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &PassiveCheckConfig{Timeout: "5m"}
	assert.Equal(t, "passive", c.CheckType())
	assert.Equal(t, "", c.GetTarget())
}

func TestMySQLCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &MySQLCheckConfig{Host: "mysql.example.com", Port: 3306}
	assert.Equal(t, "mysql", c.CheckType())
	assert.Equal(t, "mysql.example.com:3306", c.GetTarget())
}

func TestPostgreSQLCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &PostgreSQLCheckConfig{Host: "pg.example.com", Port: 5432}
	assert.Equal(t, "pgsql", c.CheckType())
	assert.Equal(t, "pg.example.com:5432", c.GetTarget())
}

func TestDNSCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &DNSCheckConfig{Domain: "example.com", RecordType: "A", Host: "8.8.8.8"}
	assert.Equal(t, "dns", c.CheckType())
	assert.Equal(t, "example.com", c.GetTarget())
}

func TestSSHCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &SSHCheckConfig{Host: "ssh.example.com", Port: 22, ExpectBanner: "SSH-2.0"}
	assert.Equal(t, "ssh", c.CheckType())
	assert.Equal(t, "ssh.example.com:22", c.GetTarget())
}

func TestRedisCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &RedisCheckConfig{Host: "redis.example.com", Port: 6379, Password: "secret", DB: 0}
	assert.Equal(t, "redis", c.CheckType())
	assert.Equal(t, "redis.example.com:6379", c.GetTarget())
}

func TestMongoDBCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &MongoDBCheckConfig{URI: "mongodb://localhost:27017/testdb"}
	assert.Equal(t, "mongodb", c.CheckType())
	assert.Equal(t, "mongodb://localhost:27017/testdb", c.GetTarget())
}

func TestMongoDBCheckConfig_EmptyURI(t *testing.T) {
	c := &MongoDBCheckConfig{}
	assert.Equal(t, "", c.GetTarget())
}

func TestDomainExpiryCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &DomainExpiryCheckConfig{Domain: "example.com", ExpiryWarningDays: 30}
	assert.Equal(t, "domain_expiry", c.CheckType())
	assert.Equal(t, "example.com", c.GetTarget())
}

func TestWebhookConfig_CheckTypeAndTarget(t *testing.T) {
	c := &WebhookConfig{
		URL:     "https://hook.example.com/notify",
		Method:  "POST",
		Payload: `{"text":"alert"}`,
		Headers: map[string]string{"Content-Type": "application/json"},
	}
	assert.Equal(t, "webhook", c.CheckType())
	assert.Equal(t, "https://hook.example.com/notify", c.GetTarget())
}

func TestSSLCertCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &SSLCertCheckConfig{Host: "example.com", Port: 443, ExpiryWarningDays: 14, ValidateChain: true}
	assert.Equal(t, "ssl_cert", c.CheckType())
	assert.Equal(t, "example.com:443", c.GetTarget())
}

func TestSMTPCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &SMTPCheckConfig{Host: "smtp.example.com", Port: 587, StartTLS: true}
	assert.Equal(t, "smtp", c.CheckType())
	assert.Equal(t, "smtp.example.com:587", c.GetTarget())
}

func TestGRPCHealthCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &GRPCHealthCheckConfig{Host: "grpc.example.com:50051", UseTLS: true}
	assert.Equal(t, "grpc_health", c.CheckType())
	assert.Equal(t, "grpc.example.com:50051", c.GetTarget())
}

func TestWebSocketCheckConfig_CheckTypeAndTarget(t *testing.T) {
	c := &WebSocketCheckConfig{URL: "wss://ws.example.com/feed"}
	assert.Equal(t, "websocket", c.CheckType())
	assert.Equal(t, "wss://ws.example.com/feed", c.GetTarget())
}

func TestAuthConfig_JSONRoundTrip(t *testing.T) {
	a := AuthConfig{User: "admin", Password: "pass123"}
	data, err := json.Marshal(a)
	assert.NoError(t, err)

	var got AuthConfig
	err = json.Unmarshal(data, &got)
	assert.NoError(t, err)
	assert.Equal(t, a.User, got.User)
	assert.Equal(t, a.Password, got.Password)
}

func TestAuthConfig_OmitEmpty(t *testing.T) {
	a := AuthConfig{}
	data, err := json.Marshal(a)
	assert.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	assert.NoError(t, err)
	// With omitempty, empty strings should be omitted
	_, hasUser := raw["user"]
	assert.False(t, hasUser)
}

func TestCheckConfig_GetTarget_TableDriven(t *testing.T) {
	tests := []struct {
		name   string
		config CheckConfig
		want   string
	}{
		{"HTTP", &HTTPCheckConfig{URL: "https://x.com"}, "https://x.com"},
		{"TCP", &TCPCheckConfig{Host: "h", Port: 1}, "h:1"},
		{"ICMP", &ICMPCheckConfig{Host: "h"}, "h"},
		{"Passive", &PassiveCheckConfig{}, ""},
		{"MySQL", &MySQLCheckConfig{Host: "h", Port: 2}, "h:2"},
		{"PostgreSQL", &PostgreSQLCheckConfig{Host: "h", Port: 3}, "h:3"},
		{"DNS", &DNSCheckConfig{Domain: "d.com"}, "d.com"},
		{"SSH", &SSHCheckConfig{Host: "h", Port: 22}, "h:22"},
		{"Redis", &RedisCheckConfig{Host: "h", Port: 4}, "h:4"},
		{"MongoDB", &MongoDBCheckConfig{URI: "mongo://h"}, "mongo://h"},
		{"DomainExpiry", &DomainExpiryCheckConfig{Domain: "d.com"}, "d.com"},
		{"Webhook", &WebhookConfig{URL: "https://w.com"}, "https://w.com"},
		{"SSLCert", &SSLCertCheckConfig{Host: "h", Port: 443}, "h:443"},
		{"SMTP", &SMTPCheckConfig{Host: "h", Port: 25}, "h:25"},
		{"GRPC", &GRPCHealthCheckConfig{Host: "h:50051"}, "h:50051"},
		{"WebSocket", &WebSocketCheckConfig{URL: "wss://w"}, "wss://w"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.config.GetTarget())
		})
	}
}
