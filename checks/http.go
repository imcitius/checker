package check

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"my/checker/config"
	projects "my/checker/projects"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

func init() {
	Checks["http"] = func(c *config.Check, p *projects.Project) error {
		var (
			answerPresent bool = true
			checkNum      int
			checkErr      error
			errorHeader   string
			tlsConfig     tls.Config
			SslExpTimeout time.Duration
			err           error
		)

		if c.AnswerPresent == "absent" {
			answerPresent = false
		} else {
			answerPresent = true
		}

		if p != nil && p.Parameters.SSLExpirationPeriod != "" {
			SslExpTimeout, err = time.ParseDuration(p.Parameters.SSLExpirationPeriod)
			if err != nil {
				config.Log.Errorf("cannot parse ssl expiration timeout: %s in project %s", err, p.Name)
			}
		}

		errorHeader = fmt.Sprintf("HTTP error at project: %s\nCheck URL: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		config.Log.Debugf("test: %s\n", c.Host)
		_, err = url.Parse(c.Host)
		if err != nil {
			config.Log.Errorf("Cannot parse http check url: %s", err)
		}
		checkNum++

		if c.SkipCheckSSL {
			tlsConfig.InsecureSkipVerify = true
		}

		// for advanced http client config we need transport
		transport := &http.Transport{}
		transport.TLSClientConfig = &tlsConfig

		client := &http.Client{Transport: transport}
		client.Timeout, _ = time.ParseDuration(c.Timeout)
		if c.StopFollowRedirects {
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return errors.New("asked to stop redirects")
			}
		}

		req, err := http.NewRequest("GET", c.Host, nil)
		if err != nil {
			return fmt.Errorf("cannot construct http request: %s", err)
		}
		if c.Auth.User != "" {
			req.SetBasicAuth(c.Auth.User, c.Auth.Password)
		}
		// if custom headers requested
		if c.Headers != nil {
			for _, headers := range c.Headers {
				for header, value := range headers {
					req.Header.Add(header, value)
				}
			}
		}

		if c.Cookies != nil {
			for _, cookie := range c.Cookies {
				req.AddCookie(&cookie)
			}
		}

		config.Log.Debugf("http request: %v", req)
		response, err := client.Do(req)

		if err != nil {
			errorMessage := errorHeader + fmt.Sprintf("answer error: %+v", err)
			return errors.New(errorMessage)
		}

		switch GetCheckScheme(c) {
		case "https":
			//config.Log.Debugf("SSL: %v", response.TLS.PeerCertificates)
			if len(response.TLS.PeerCertificates) > 0 {
				for i, cert := range response.TLS.PeerCertificates {
					if time.Until(cert.NotAfter) < SslExpTimeout {
						config.Log.Infof("Cert #%d subject: %s, NotBefore: %v, NotAfter: %v", i, cert.Subject, cert.NotBefore, cert.NotAfter)
					}
					config.Log.Debugf("server TLS: %+v", response.TLS.PeerCertificates[i].NotAfter)
				}
			} else {
				errorMessage := errorHeader + "No certificates present on https connection"
				return errors.New(errorMessage)
			}
		}

		if response.Body != nil {
			defer response.Body.Close()
		} else {
			errorMessage := errorHeader + fmt.Sprintf("empty body: %+v", response)
			return errors.New(errorMessage)
		}
		//config.Log.Printf("4")

		buf := new(bytes.Buffer)
		buf.ReadFrom(response.Body)
		//config.Log.Printf("Server: %s, http answer body: %s\n", c.Host, buf)
		// check that response code is correct

		// found actual return code in answer codes slice
		checkCode := func(codes []int, answercode int) bool {
			found := false
			// init asnwer codes slice if empty
			if len(codes) == 0 {
				codes = []int{200}
			}
			for _, c := range codes {
				if c == answercode {
					found = true
				}
			}
			return found
		}(c.Code, int(response.StatusCode))

		if !checkCode {
			errorMessage := errorHeader + fmt.Sprintf("HTTP response code error: %d (want %d)", response.StatusCode, c.Code)
			return errors.New(errorMessage)
		}

		answer, _ := regexp.Match(c.Answer, buf.Bytes())
		// check answer_present condition
		answerGood := (answer == answerPresent) && checkCode
		//config.Log.Printf("Answer: %v, AnswerPresent: %v, AnswerGood: %v", answer, c.AnswerPresent, answerGood)

		if !answerGood {
			errorMessage := errorHeader + fmt.Sprintf("answer text error: found '%s' ('%s' should be %s)", buf.String(), c.Answer, c.AnswerPresent)
			return errors.New(errorMessage)
		}

		return checkErr

	}
}
