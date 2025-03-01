package checks

import (
	"time"
)

const (
	ErrEmptyHost          = "host is empty"
	ErrICMPError          = "ICMP error: %s"
	ErrPacketLoss         = "ping error: %f percent packet loss"
	ErrOther              = "other ping error: %s"
	ErrEmptyPort          = "port is empty"
	ErrCannotParseTimeout = "%scannot parse http check timeout: %s"

	// HTTP errors
	ErrCantBuildHTTPRequest   = "cannot construct http request"
	ErrGotRedirect            = "asked to stop redirects"
	ErrHTTPClientConstruction = "http client construction error"
	ErrHTTPURLParse           = "failed to parse URL"
	ErrHTTPRequestError       = "HTTP error"
	ErrHTTPNoSSL              = "no SSL certificates present on https connection"
	ErrParseSSlTimeout        = "invalid SSL expiration period"
	InfoSSLCertDetails        = "Info: Certificate #%d subject: %s, NotBefore: %v, NotAfter: %v"

	ErrHTTPAnswerError       = "answer error: %+v, timeout %s"
	ErrHTTPEmptyBody         = "empty response body: %+v"
	ErrHTTPBodyRead          = "%serror reading answer body: %v (%d bytes)"
	ErrHTTPResponseCodeError = "%s HTTP response code error: %d (want %d)"
	ErrHTTPAnswerTextError   = "%sanswer text error: expected pattern '%s' not found in response: '%s'"
	ErrHTTPRegexParseError   = "%serror processing answer regex: %v"
	ErrHTTPAnswerTooLong     = "answer is %d bytes long, check the logs"

	ErrTCPError = "TCP error"

	ErrGetFile          = "File get error "
	ErrEmptyUrl         = "Url is empty"
	ErrEmptyHash        = "Hash of empty file"
	ErrFileReadOnly     = "Temp file '%s' appears to be read-only"
	ErrOpenTempFile     = "Can't open temp file"
	ErrCloseTempFile    = "Error closing temp file: '%s'"
	ErrCheckWritable    = "Temp file is not writable"
	ErrDownload         = "Error downloading file, error: '%s', code: '%d'"
	ErrCantReadFile     = "Cannot read downloaded file %s: %s"
	ErrFileSizeMismatch = "File size mismatch: config size %d, downloaded size: %d"
	ErrFileHashMismatch = "File hash mismatch: config hash %s, downloaded hash: %s"
)

// Checker is an interface that all health checks should implement.
// It defines a universal Run method.
type Checker interface {
	// Run executes the health check and returns:
	// - a bool indicating if the check passed, and
	// - a message detailing the result.
	Run() (time.Duration, error)
}
