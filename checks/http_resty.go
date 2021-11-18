package check

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"my/checker/config"
	projects "my/checker/projects"
	"net/url"
	"regexp"
	"time"
)

func init() {
	Checks["http"] = func(c *config.Check, p *projects.Project) error {
		var (
			answerPresent bool
			checkNum      int
			checkErr      error
			errorHeader   string
			tlsConfig     tls.Config
			SslExpTimeout time.Duration
			err           error
		)

		if c.AnswerPresent != "absent" {
			answerPresent = true
		}

		if p != nil && p.Parameters.SSLExpirationPeriod != "" {
			SslExpTimeout, err = time.ParseDuration(p.Parameters.SSLExpirationPeriod)
			if err != nil {
				config.Log.Errorf("cannot parse ssl expiration timeout: %s in project %s", err, p.Name)
			}
		}

		errorHeader = fmt.Sprintf("HTTP error at project: %s\nCheck URL: %s\nCheck UUID: %s\nCheck name: %s\n", p.Name, c.Host, c.UUid, c.Name)

		config.Log.Debugf("test: %s\n", c.Host)
		_, err = url.Parse(c.Host)
		if err != nil {
			config.Log.Errorf("Cannot parse http check url: %s", err)
		}
		checkNum++

		if c.SkipCheckSSL {
			tlsConfig.InsecureSkipVerify = true
		}

		timeout, err := time.ParseDuration(c.Timeout)
		if err != nil {
			config.Log.Errorf("cannot parse http check timeout: %s in project %s, check %s", err.Error(), p.Name, c.Host)
		}
		client := resty.New()
		client.SetTLSClientConfig(&tlsConfig)
		client.SetTimeout(timeout)
		if c.Auth.User != "" {
			client.SetBasicAuth(c.Auth.User, c.Auth.Password)
		}
		if c.Headers != nil {
			for _, headers := range c.Headers {
				for header, value := range headers {
					client.Header.Add(header, value)
				}
			}
		}
		if c.Cookies != nil {
			for _, cookie := range c.Cookies {
				client.SetCookie(&cookie)
			}
		}

		response, err := client.R().
			EnableTrace().
			Get(c.Host)
		if err != nil {
			errorMessage := errorHeader + "HTTP request error\n" + err.Error() + "\n"
			config.Log.Infof(errorMessage)
			return errors.New(errorMessage)
		}

		switch c.GetCheckScheme() {
		case "https":
			config.Log.Infof("SSL: %+v", response.RawResponse.TLS.PeerCertificates)

			if len(response.RawResponse.TLS.PeerCertificates) > 0 {
				err := checkCertificatesExpiration(response, SslExpTimeout)
				if err != nil {
					errorMessage := errorHeader + "TLS Certificate will expire too soon\n" + err.Error() + "\n"
					config.Log.Infof(errorMessage)
					return errors.New(errorMessage)
				}
			}
		}
		if response.Body() == nil {
			//	defer func() { _ = response.Body.Close() }()
			//} else {
			errorMessage := errorHeader + fmt.Sprintf("empty response body: %+v", response)
			return errors.New(errorMessage)
		}
		checkCode := func(codes []int, answercode int) bool {
			found := false
			// init answer codes slice if empty
			if len(codes) == 0 {
				codes = []int{200}
			}
			for _, c := range codes {
				if c == answercode {
					found = true
				}
			}
			return found
		}(c.Code, response.StatusCode())
		if !checkCode {
			errorMessage := errorHeader + fmt.Sprintf("HTTP response code error: %d (want %d)", response.StatusCode, c.Code)
			return errors.New(errorMessage)
		}

		answer, _ := regexp.Match(c.Answer, response.Body())
		// check answer_present condition
		answerGood := (answer == answerPresent) && checkCode
		//config.Log.Printf("Answer: %v, AnswerPresent: %v, AnswerGood: %v", answer, c.AnswerPresent, answerGood)

		if !answerGood {
			answer := response.Body()
			if len(response.Body()) > 350 {
				answer = []byte("Answer is too long, check the logs")
			}
			errorMessage := errorHeader + fmt.Sprintf("answer text error: found '%s' ('%s' should be %s)", answer, c.Answer, c.AnswerPresent)
			return errors.New(errorMessage)
		}
		config.Log.Infof("%+v", response.Request.TraceInfo())

		return checkErr
	}
}

func checkCertificatesExpiration(response *resty.Response, exp time.Duration) error {
	for i, cert := range response.RawResponse.TLS.PeerCertificates {
		if time.Until(cert.NotAfter) < exp {
			err := fmt.Errorf("Cert #%d subject: %s, NotBefore: %v, NotAfter: %v", i, cert.Subject, cert.NotBefore, cert.NotAfter)
			config.Log.Infof(err.Error())
			return err
		}
		config.Log.Debugf("server TLS: %+v", response.RawResponse.TLS.PeerCertificates[i].NotAfter)
	}
	return nil
}
