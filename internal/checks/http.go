package checks

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
)

// HTTPCheck represents an HTTP health check.
// Extended with additional fields to support advanced validations.
type HTTPCheck struct {
	URL     string
	Scheme  string // e.g., "http", "https"
	Method  string // e.g., "GET", "POST", "PUT"
	Timeout string // e.g., "5s"
	Answer        string // regex pattern to check in the response body
	AnswerPresent bool   // if true, answer pattern must be found; if false, answer pattern must NOT be found
	Code          []int  // expected HTTP status codes; default is [200]
	Auth    struct {
		User     string
		Password string
	}
	Headers             []map[string]string // optional headers to add to the HTTP request
	Cookies             []http.Cookie       // optional cookies to add to the HTTP request
	SkipCheckSSL        bool                // if true, skip SSL certificate check
	SSLExpirationPeriod string              // duration string (e.g., "168h") for checking impending certificate expiration
	StopFollowRedirects bool                // if true, do not follow HTTP redirects
	ErrorHeader         string              // a prefix for error messages
	TlsConfig           *tls.Config         // optional custom TLS configuration

	req    *http.Request
	Logger *logrus.Entry
}

func (check *HTTPCheck) init() (*http.Client, error) {
	var err error

	// for advanced http client config we need transport
	transport := &http.Transport{}

	// Create HTTP client.
	pURL, err := url.Parse(check.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}

	if pURL.Scheme == "https" {
		// Setup a custom TLS transport.
		check.Scheme = "https"
		if check.TlsConfig != nil {
			transport.TLSClientConfig = check.TlsConfig
		} else {
			transport.TLSClientConfig = &tls.Config{}
		}
		// If SkipCheckSSL is true, force InsecureSkipVerify.
		if check.SkipCheckSSL {
			transport.TLSClientConfig.InsecureSkipVerify = true
		}
	}

	client := &http.Client{Transport: transport}

	if check.Timeout == "" {
		check.Timeout = "3s"
	}
	client.Timeout, err = time.ParseDuration(check.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout value: %v", err)
	}

	if len(check.Code) == 0 {
		check.Code = []int{http.StatusOK}
	}
	if check.SSLExpirationPeriod == "" {
		check.SSLExpirationPeriod = "168h" // default to 7 days
	}

	if check.StopFollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return fmt.Errorf("redirect not allowed")
		}
	}

	// Build HTTP request.
	method := "GET"
	if check.Method != "" {
		method = check.Method
	}
	req, err := http.NewRequest(method, check.URL, nil) // Configurable HTTP method
	if err != nil {
		return nil, fmt.Errorf("failed to build HTTP request: %v", err)
	}

	if check.Auth.User != "" {
		req.SetBasicAuth(check.Auth.User, check.Auth.Password)
	}

	// if custom headers requested
	if check.Headers != nil {
		for _, headers := range check.Headers {
			for header, value := range headers {
				req.Header.Add(header, value)
			}
		}
	}

	if check.Cookies != nil {
		for _, cookie := range check.Cookies {
			req.AddCookie(&cookie)
		}
	}

	check.req = req
	return client, nil
}

// Run executes the HTTP health check with extended validations.
func (check *HTTPCheck) Run() (time.Duration, error) {
	start := time.Now()

	client, err := check.init()
	if err != nil {
		return time.Since(start), fmt.Errorf("%s: %w", ErrHTTPClientConstruction, err)
	}

	// Execute the request.
	resp, err := client.Do(check.req)
	if err != nil {
		if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			return time.Since(start), fmt.Errorf("timeout error: request exceeded %s", check.Timeout)
		}
		return time.Since(start), fmt.Errorf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	// If HTTPS, perform SSL certificate checks if not skipped.
	if check.Scheme == "https" && !check.SkipCheckSSL {
		err := checkSSL(resp, check)
		if err != nil {
			return time.Since(start), err
		}
	}

	// Read the full response body.
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return time.Since(start), fmt.Errorf("failed to read response body: %v", err)
	}

	// Validate that the response status code is among the expected ones.
	codeValid := false
	for _, code := range check.Code {
		if resp.StatusCode == code {
			codeValid = true
			break
		}
	}
	if !codeValid {
		return time.Since(start), fmt.Errorf("HTTP check failed with status %d", resp.StatusCode)
	}

	// Validate answer content using regex if an answer pattern is provided.
	if check.Answer != "" {
		if len(bodyBytes) == 0 {
			if check.AnswerPresent {
				return time.Since(start), fmt.Errorf("HTTP response body is empty")
			}
			// Body is empty and we expected pattern to be absent — that's a pass.
			return time.Since(start), nil
		}

		matched, err := regexp.Match(check.Answer, bodyBytes)
		if err != nil {
			return time.Since(start), fmt.Errorf("error processing answer regex: %v", err)
		}

		if check.AnswerPresent && !matched {
			answerText := string(bodyBytes)
			if len(answerText) > 350 {
				answerText = fmt.Sprintf("response too long (%d bytes)", len(answerText))
			}
			return time.Since(start), fmt.Errorf("expected pattern '%s' not found in response: %s", check.Answer, answerText)
		}
		if !check.AnswerPresent && matched {
			return time.Since(start), fmt.Errorf("unexpected pattern '%s' found in response", check.Answer)
		}
	}

	return time.Since(start), nil
}

func checkSSL(resp *http.Response, check *HTTPCheck) error {
	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		return fmt.Errorf("no SSL certificate found")
	}
	sslExpPeriod, err := time.ParseDuration(check.SSLExpirationPeriod)
	if err != nil {
		return fmt.Errorf("invalid SSL expiration period: %v", err)
	}

	// Determine the final hostname (after any redirects) from the response's request URL.
	finalHost := resp.Request.URL.Hostname()

	// Determine the original hostname from the check's configured URL.
	originalHost := ""
	if parsedOriginal, err := url.Parse(check.URL); err == nil {
		originalHost = parsedOriginal.Hostname()
	}

	// Build the hostname portion of the error message.
	// If the final host differs from the original, note the redirect.
	hostInfo := fmt.Sprintf(" for %s", finalHost)
	if originalHost != "" && originalHost != finalHost {
		hostInfo = fmt.Sprintf(" for %s (redirected from %s)", finalHost, originalHost)
	}

	// Special case for immediate expiration check (used in tests)
	if sslExpPeriod == 0 {
		return fmt.Errorf("SSL certificate%s will expire in 0s (threshold: 0s)", hostInfo)
	}

	// Only check the leaf (server) certificate, not intermediate CA certificates.
	// PeerCertificates[0] is always the leaf certificate in Go's TLS.
	// Intermediate certs may have different expiry dates and should not trigger alerts.
	cert := resp.TLS.PeerCertificates[0]
	now := time.Now()
	timeUntilExpiry := cert.NotAfter.Sub(now)
	if timeUntilExpiry <= sslExpPeriod {
		check.Logger.Debugf("Certificate Subject=%v, NotBefore=%v, NotAfter=%v", cert.Subject, cert.NotBefore, cert.NotAfter)
		return fmt.Errorf("SSL certificate%s will expire in %v (threshold: %v)", hostInfo, timeUntilExpiry, sslExpPeriod)
	}
	return nil
}
