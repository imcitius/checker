package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

func runHTTPCheck(c *Check, p *Project) error {
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
		log.Fatal(err)
	}

	errorHeader = fmt.Sprintf("HTTP error at project: %s\nCheck URL: %s\nCheck UUID: %s\n", p.Name, c.Host, c.uuID)

	log.Debugf("test: %s\n", c.Host)
	_, err = url.Parse(c.Host)
	if err != nil {
		log.Fatal(err)
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
			req.AddCookie(cookie)
		}
	}

	log.Debugf("http request: %v", req)
	response, err := client.Do(req)

	if c.GetScheme() == "https" {
		for i, cert := range response.TLS.PeerCertificates {
			if cert.NotAfter.Sub(time.Now()) < sslExpTimeout {
				log.Infof("Cert #%d subject: %s, NotBefore: %v, NotAfter: %v", i, cert.Subject, cert.NotBefore, cert.NotAfter)
			}
			log.Debugf("server TLS: %+v", response.TLS.PeerCertificates[i].NotAfter)
		}

	}
	//log.Printf("2")

	if err != nil {
		errorMessage := errorHeader + fmt.Sprintf("asnwer error: %+v", err)
		return errors.New(errorMessage)
	}
	//log.Printf("3")

	if response.Body != nil {
		defer response.Body.Close()
	} else {
		errorMessage := errorHeader + fmt.Sprintf("empty body: %+v", response)
		return errors.New(errorMessage)
	}
	//log.Printf("4")

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	//log.Printf("Server: %s, http answer body: %s\n", c.Host, buf)
	// check that response code is correct

	if c.Code == 0 {
		c.Code = 200
	}
	code := c.Code == int(response.StatusCode)
	if !code {
		errorMessage := errorHeader + fmt.Sprintf("HTTP response code error: %d (want %d)", response.StatusCode, c.Code)
		return errors.New(errorMessage)
	}

	answer, _ := regexp.Match(c.Answer, buf.Bytes())
	// check answer_present condition
	answerGood := (answer == answerPresent) && code
	//log.Printf("Answer: %v, AnswerPresent: %v, AnswerGood: %v", answer, c.AnswerPresent, answerGood)

	if !answerGood {
		errorMessage := errorHeader + fmt.Sprintf("answer text error: found '%s' ('%s' should be %s)", string(buf.Bytes()), c.Answer, c.AnswerPresent)
		return errors.New(errorMessage)
	}

	return checkErr
}
