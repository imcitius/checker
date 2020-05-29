package check

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"my/checker/config"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

func init() {
	config.Checks["http"] = func(c *config.Check, p *config.Project) error {
		var (
			answerPresent bool = true
			checkNum      int
			checkErr      error
			errorHeader   string
			tlsConfig     tls.Config
		)

		if c.AnswerPresent == "absent" {
			answerPresent = false
		} else {
			answerPresent = true
		}

		sslExpTimeout, err := time.ParseDuration(p.Parameters.SSLExpirationPeriod)
		if err != nil {
			config.Log.Fatal(err)
		}

		errorHeader = fmt.Sprintf("HTTP error at project: %s\nCheck URL: %s\nCheck UUID: %s\n", p.Name, c.Host, c.UUid)

		config.Log.Debugf("test: %s\n", c.Host)
		_, err = url.Parse(c.Host)
		if err != nil {
			config.Log.Fatal(err)
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
				return errors.New("Asked to stop redirects")
			}
		}

		req, err := http.NewRequest("GET", c.Host, nil)
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

		if GetCheckScheme(c) == "https" {
			//config.Log.Debugf("SSL: %v", response.TLS.PeerCertificates)
			if len(response.TLS.PeerCertificates) > 0 {
				for i, cert := range response.TLS.PeerCertificates {
					if cert.NotAfter.Sub(time.Now()) < sslExpTimeout {
						config.Log.Infof("Cert #%d subject: %s, NotBefore: %v, NotAfter: %v", i, cert.Subject, cert.NotBefore, cert.NotAfter)
					}
					config.Log.Debugf("server TLS: %+v", response.TLS.PeerCertificates[i].NotAfter)
				}
			} else {
				errorMessage := errorHeader + fmt.Sprint("No certificates present on https connection")
				return errors.New(errorMessage)
			}

		}
		//config.Log.Printf("2")

		if err != nil {
			errorMessage := errorHeader + fmt.Sprintf("asnwer error: %+v", err)
			return errors.New(errorMessage)
		}
		//config.Log.Printf("3")

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

		// init asnwer codes slice if empty
		if len(c.Code) == 0 {
			c.Code = []int{200}
		}
		// found actual return code in answer codes slice
		code := func(codes []int, answercode int) bool {
			found := false
			for _, c := range codes {
				if c == answercode {
					found = true
				}
			}
			return found
		}(c.Code, int(response.StatusCode))

		if !code {
			errorMessage := errorHeader + fmt.Sprintf("HTTP response code error: %d (want %d)", response.StatusCode, c.Code)
			return errors.New(errorMessage)
		}

		answer, _ := regexp.Match(c.Answer, buf.Bytes())
		// check answer_present condition
		answerGood := (answer == answerPresent) && code
		//config.Log.Printf("Answer: %v, AnswerPresent: %v, AnswerGood: %v", answer, c.AnswerPresent, answerGood)

		if !answerGood {
			errorMessage := errorHeader + fmt.Sprintf("answer text error: found '%s' ('%s' should be %s)", string(buf.Bytes()), c.Answer, c.AnswerPresent)
			return errors.New(errorMessage)
		}

		return checkErr

	}
}
