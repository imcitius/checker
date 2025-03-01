package checks

import (
	"crypto/tls"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// HTTPCheck represents an HTTP health check.
// Extended with additional fields to support advanced validations.
type HTTPCheck struct {
	URL     string
	Scheme  string // e.g., "http", "https"
	Timeout string // e.g., "5s"
	Answer  string // regex pattern to check in the response body
	Code    []int  // expected HTTP status codes; default is [200]
	Auth    struct {
		User     string
		Password string
	}
	Headers             []map[string]string // optional headers to add to the HTTP request
	Cookies             []http.Cookie       // optional cookies to add to the HTTP request
	SkipCheckSSL        bool                // if true, skip SSL certificate check
	SSLExpirationPeriod string              // duration string (e.g., "720h") for checking impending certificate expiration
	StopFollowRedirects bool                // if true, do not follow HTTP redirects
	ErrorHeader         string              // a prefix for error messages
	TlsConfig           *tls.Config         // optional custom TLS configuration

	req    *http.Request
	Logger *logrus.Entry
}

func (hc *HTTPCheck) init() (*http.Client, error) {
	var err error

	// for advanced http client config we need transport
	transport := &http.Transport{}

	// Create HTTP client.
	pURL, err := url.Parse(hc.URL)
	if err != nil {
		return nil, fmt.Errorf(ErrHTTPURLParse, err)
	}

	if pURL.Scheme == "https" {
		// Setup a custom TLS transport.
		hc.Scheme = "https"
		if hc.TlsConfig != nil {
			transport.TLSClientConfig = hc.TlsConfig
		} else {
			transport.TLSClientConfig = &tls.Config{}
		}
		// If SkipCheckSSL is true, force InsecureSkipVerify.
		if hc.SkipCheckSSL {
			transport.TLSClientConfig.InsecureSkipVerify = true
		}
	}

	client := &http.Client{Transport: transport}

	if hc.Timeout == "" {
		hc.Timeout = "3s"
	}
	client.Timeout, err = time.ParseDuration(hc.Timeout)
	if err != nil {
		return nil, fmt.Errorf(ErrCannotParseTimeout, hc.Timeout)
	}

	if len(hc.Code) == 0 {
		hc.Code = []int{http.StatusOK}
	}
	if hc.SSLExpirationPeriod == "" {
		hc.SSLExpirationPeriod = "720h" // default to 30 days
	}

	if hc.StopFollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return fmt.Errorf(ErrGotRedirect)
		}
	}

	// Build HTTP request.
	req, err := http.NewRequest("GET", hc.URL, nil) // TODO add more HTTP methods
	if err != nil {
		return nil, fmt.Errorf(ErrCantBuildHTTPRequest, err.Error())
	}

	if hc.Auth.User != "" {
		req.SetBasicAuth(hc.Auth.User, hc.Auth.Password)
	}

	// if custom headers requested
	if hc.Headers != nil {
		for _, headers := range hc.Headers {
			for header, value := range headers {
				req.Header.Add(header, value)
			}
		}
	}

	if hc.Cookies != nil {
		for _, cookie := range hc.Cookies {
			req.AddCookie(&cookie)
		}
	}

	hc.req = req
	return client, nil
}

// Run executes the HTTP health check with extended validations.
func (hc *HTTPCheck) Run() (time.Duration, error) {
	start := time.Now()

	client, err := hc.init()
	if err != nil {
		return time.Now().Sub(start), fmt.Errorf(ErrHTTPClientConstruction, err)
	}

	// Execute the request.
	resp, err := client.Do(hc.req)
	if err != nil {
		return time.Now().Sub(start), fmt.Errorf(ErrHTTPRequestError, err)
	}
	defer resp.Body.Close()

	// If HTTPS, perform SSL certificate checks if not skipped.
	if hc.Scheme == "https" && !hc.SkipCheckSSL {
		err := checkSSL(resp, hc)
		if err != nil {
			return time.Now().Sub(start), err
		}
	}

	// Read the full response body.
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return time.Now().Sub(start), fmt.Errorf(ErrHTTPBodyRead, hc.ErrorHeader, err, len(bodyBytes))
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
		return time.Now().Sub(start), fmt.Errorf(ErrHTTPResponseCodeError, hc.ErrorHeader, resp.StatusCode, hc.Code)
	}

	// Validate answer content using regex if an answer pattern is provided.
	if hc.Answer != "" {
		if len(bodyBytes) == 0 {
			return time.Now().Sub(start), fmt.Errorf(ErrHTTPEmptyBody)
		}

		matched, err := regexp.Match(hc.Answer, bodyBytes)
		if err != nil {
			return time.Now().Sub(start), fmt.Errorf(ErrHTTPRegexParseError, hc.ErrorHeader, err)
		}

		if !matched {
			answerText := string(bodyBytes)
			if len(answerText) > 350 {
				answerText = fmt.Sprintf(ErrHTTPAnswerTooLong, len(answerText))
			}
			return time.Now().Sub(start), fmt.Errorf(ErrHTTPAnswerTextError, hc.ErrorHeader, hc.Answer, answerText)
		}
	}

	return time.Now().Sub(start), nil
}

func checkSSL(resp *http.Response, hc *HTTPCheck) error {
	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		return fmt.Errorf(ErrHTTPNoSSL)
	}
	sslExpPeriod, err := time.ParseDuration(hc.SSLExpirationPeriod)
	if err != nil {
		return fmt.Errorf(ErrParseSSlTimeout)
	}
	for i, cert := range resp.TLS.PeerCertificates {
		if time.Until(cert.NotAfter) < sslExpPeriod {
			// Log certificate details (using fmt.Printf for simplicity).
			hc.Logger.Debugf(InfoSSLCertDetails, i, cert.Subject, cert.NotBefore, cert.NotAfter)
		}
	}
	return nil
}
