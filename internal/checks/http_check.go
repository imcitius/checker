package checks

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// HTTPCheck represents an HTTP health check.
// Extended with additional fields to support advanced validations.
type HTTPCheck struct {
	URL                 string
	Timeout             string            // e.g., "5s"
	Answer              string            // regex pattern to check in the response body
	AnswerPresent       bool              // if true, answer must be present; if false, it must be absent
	Code                []int             // expected HTTP status codes; default is [200]
	Headers             map[string]string // optional headers to add to the HTTP request
	SkipCheckSSL        bool              // if true, skip SSL certificate check
	SSLExpirationPeriod string            // duration string (e.g., "720h") for checking impending certificate expiration
	StopFollowRedirects bool              // if true, do not follow HTTP redirects
	ErrorHeader         string            // a prefix for error messages
	TlsConfig           *tls.Config       // optional custom TLS configuration
}

// Run executes the HTTP health check with extended validations.
func (hc *HTTPCheck) Run() (bool, string) {
	start := time.Now()

	// Set default values if necessary.
	if hc.Timeout == "" {
		hc.Timeout = "5s"
	}
	timeout, err := time.ParseDuration(hc.Timeout)
	if err != nil {
		return false, fmt.Sprintf("%sinvalid timeout value: %v", hc.ErrorHeader, err)
	}
	if len(hc.Code) == 0 {
		hc.Code = []int{http.StatusOK}
	}
	if hc.SSLExpirationPeriod == "" {
		hc.SSLExpirationPeriod = "720h" // default to 30 days
	}
	if hc.ErrorHeader == "" {
		hc.ErrorHeader = "HTTP check error: "
	}

	// Parse the URL to detect the scheme.
	pURL, err := url.Parse(hc.URL)
	if err != nil {
		return false, fmt.Sprintf("%sfailed to parse URL: %v", hc.ErrorHeader, err)
	}

	// Create HTTP client.
	var client *http.Client
	if pURL.Scheme == "https" {
		// Setup a custom TLS transport.
		tr := &http.Transport{}
		if hc.TlsConfig != nil {
			tr.TLSClientConfig = hc.TlsConfig
		} else {
			tr.TLSClientConfig = &tls.Config{}
		}
		// If SkipCheckSSL is true, force InsecureSkipVerify.
		if hc.SkipCheckSSL {
			tr.TLSClientConfig.InsecureSkipVerify = true
		}
		client = &http.Client{
			Timeout:   timeout,
			Transport: tr,
		}
	} else {
		client = &http.Client{Timeout: timeout}
	}
	if hc.StopFollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return errors.New("redirect not allowed")
		}
	}

	// Build HTTP request.
	req, err := http.NewRequest("GET", hc.URL, nil)
	if err != nil {
		return false, fmt.Sprintf("%sfailed to build HTTP request: %v", hc.ErrorHeader, err)
	}
	// Add custom headers if provided.
	for key, value := range hc.Headers {
		req.Header.Add(key, value)
	}

	// Execute the request.
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("%sHTTP error: %v", hc.ErrorHeader, err)
	}
	defer resp.Body.Close()

	// If HTTPS, perform SSL certificate checks if not skipped.
	if pURL.Scheme == "https" && !hc.SkipCheckSSL {
		if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
			return false, fmt.Sprintf("%sno SSL certificates present on https connection", hc.ErrorHeader)
		}
		sslExpPeriod, err := time.ParseDuration(hc.SSLExpirationPeriod)
		if err != nil {
			return false, fmt.Sprintf("%sinvalid SSL expiration period: %v", hc.ErrorHeader, err)
		}
		for i, cert := range resp.TLS.PeerCertificates {
			if time.Until(cert.NotAfter) < sslExpPeriod {
				// Log certificate details (using fmt.Printf for simplicity).
				fmt.Printf("Info: Certificate #%d subject: %s, NotBefore: %v, NotAfter: %v\n", i, cert.Subject, cert.NotBefore, cert.NotAfter)
			}
		}
	}

	// Read the full response body.
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Sprintf("%serror reading response body: %v", hc.ErrorHeader, err)
	}

	// Validate that the response status code is among the expected ones.
	codeValid := false
	for _, code := range hc.Code {
		if resp.StatusCode == code {
			codeValid = true
			break
		}
	}
	if !codeValid {
		return false, fmt.Sprintf("%sHTTP check failed with status %d, expected %v", hc.ErrorHeader, resp.StatusCode, hc.Code)
	}

	// Validate answer content using regex if an answer pattern is provided.
	if hc.Answer != "" {
		matched, err := regexp.Match(hc.Answer, bodyBytes)
		if err != nil {
			return false, fmt.Sprintf("%serror processing answer regex: %v", hc.ErrorHeader, err)
		}
		if hc.AnswerPresent && !matched {
			answerText := string(bodyBytes)
			if len(answerText) > 350 {
				answerText = "Answer is too long, check the logs"
			}
			return false, fmt.Sprintf("%sanswer text error: expected pattern '%s' not found in response: '%s'", hc.ErrorHeader, hc.Answer, answerText)
		}
		if !hc.AnswerPresent && matched {
			answerText := string(bodyBytes)
			if len(answerText) > 350 {
				answerText = "Answer is too long, check the logs"
			}
			return false, fmt.Sprintf("%sanswer text error: pattern '%s' was found in response but should be absent. Response: '%s'", hc.ErrorHeader, hc.Answer, answerText)
		}
	}

	duration := time.Since(start)
	return true, fmt.Sprintf("HTTP check passed in %v", duration)
}
