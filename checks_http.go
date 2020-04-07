package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
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

	errorHeader = fmt.Sprintf("HTTP error at project: %s\nCheck URL: %s\nCheck UUID: %s\n", p.Name, c.Host, c.uuID)

	fmt.Printf("test: %s\n", c.Host)
	_, err := url.Parse(c.Host)
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
	client.Timeout = c.Timeout * time.Millisecond // milliseconds
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

	// log.Printf("http request: %v", req)
	response, err := client.Do(req)

	for i, cert := range response.TLS.PeerCertificates {
		if cert.NotAfter.Sub(time.Now()) < 720*time.Hour {
			log.Printf("Cert #%d subject: %s, NotBefore: %v, NotAfter: %v", i, cert.Subject, cert.NotBefore, cert.NotAfter)
		}
		//log.Printf("server TLS: %+v", response.TLS.PeerCertificates[i].NotAfter)
	}

	if err != nil {
		errorMessage := errorHeader + fmt.Sprintf("asnwer error: %+v", err)
		return errors.New(errorMessage)
	}

	if response.Body != nil {
		defer response.Body.Close()
	} else {
		errorMessage := errorHeader + fmt.Sprintf("empty body: %+v", response)
		return errors.New(errorMessage)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	//log.Printf("Server: %s, http answer body: %s\n", c.URL, buf)
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
