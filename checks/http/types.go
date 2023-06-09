package http

import (
	"crypto/tls"
	"net/http"
	"time"
)

type IHTTPCheck interface {
	RealExecute() (time.Duration, error)
}

type THTTPCheck struct {
	Project   string
	CheckName string

	Url           string
	Timeout       string
	Answer        string
	AnswerPresent string `mapstructure:"answer_present"`
	Code          []int
	Auth          struct {
		User     string
		Password string
	}
	Headers                   []map[string]string
	Cookies                   []http.Cookie
	SkipCheckSSL              bool `mapstructure:"skip_check_ssl"`
	SSLExpirationPeriod       string
	SSLExpirationPeriodParsed time.Duration
	StopFollowRedirects       bool `mapstructure:"stop_follow_redirects"`

	ErrorHeader string
	req         *http.Request
	Scheme      string
	TlsConfig   tls.Config
}

const (
	ErrWrongCheckType         = "Wrong check type: %s (should be http)"
	ErrCannotParseTimeout     = "cannot parse http check timeout: %s in project %s, check %s"
	ErrGotRedirect            = "asked to stop redirects: project %s, check %s"
	ErrCantBuildHTTPRequest   = "cannot construct http request: project %s, check %s"
	ErrHTTPClientConstruction = "Http client construction error: %+v"
	ErrHTTPAnswerError        = "answer error: %+v, timeout %s"
	ErrHTTPNoCertificates     = "No certificates present on https connection"
	ErrHTTPEmptyBody          = "empty response body: %+v"
	ErrHTTPBodyRead           = "Error reading answer body: %s (length %d)"
	ErrHTTPResponseCodeError  = "HTTP response code error: %d (want %d)"
	ErrHTTPAnswerTextError    = "answer text error: found '%s' ('%s' should be %s)"
)
