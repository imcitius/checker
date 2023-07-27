package http

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

func (c *THTTPCheck) init() (*http.Client, error) {
	var err error

	// for advanced http client config we need transport
	transport := &http.Transport{}
	transport.TLSClientConfig = &c.TlsConfig

	client := &http.Client{Transport: transport}
	client.Timeout, _ = time.ParseDuration(c.Timeout)
	if err != nil {
		return nil, errors.New(fmt.Sprintf(ErrCannotParseTimeout, err.Error(), "project", c.Url))
	}

	if c.StopFollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return errors.New(ErrGotRedirect)
		}
	}

	req, err := http.NewRequest("GET", c.Url, nil) // TODO add more HTTP methods
	if err != nil {
		return nil, errors.New(fmt.Sprintf(ErrCantBuildHTTPRequest))
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

	c.req = req
	return client, nil
}

func (c *THTTPCheck) RealExecute() (time.Duration, error) {
	var err error

	start := time.Now()

	client, err := c.init()

	if err != nil {
		errorMessage := c.ErrorHeader + fmt.Sprintf(ErrHTTPClientConstruction, err)
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	logger.Debugf("http request: %v", c.req)
	response, err := client.Do(c.req)
	if err != nil {
		errorMessage := c.ErrorHeader + fmt.Sprintf(ErrHTTPAnswerError, err, c.Timeout)
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	switch c.Scheme {
	case "https":
		//logger.Debugf("SSL: %v", response.TLS.PeerCertificates)
		if len(response.TLS.PeerCertificates) > 0 {
			for i, cert := range response.TLS.PeerCertificates {
				if time.Until(cert.NotAfter) < (c.SSLExpirationPeriodParsed) {
					logger.Infof("Cert #%d subject: %s, NotBefore: %v, NotAfter: %v", i, cert.Subject, cert.NotBefore, cert.NotAfter)
				}
				logger.Debugf("server TLS: %+v", response.TLS.PeerCertificates[i].NotAfter)
			}
		} else {
			errorMessage := c.ErrorHeader + ErrHTTPNoCertificates
			return time.Now().Sub(start), errors.New(errorMessage)
		}
	}

	if response.Body != nil {
		defer func() { _ = response.Body.Close() }()
	} else {
		errorMessage := c.ErrorHeader + fmt.Sprintf(ErrHTTPEmptyBody, response)
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	buf := new(bytes.Buffer)
	if n, err := buf.ReadFrom(response.Body); err != nil {
		logger.Warnf(ErrHTTPBodyRead, err, n)
	}
	//logger.Printf("Server: %s, http answer body: %s\n", c.Host, buf)
	// check that response code is correct

	// found actual return code in answer codes slice
	checkCode := checkAnswerCode(c.Code, response.StatusCode)

	if !checkCode {
		errorMessage := c.ErrorHeader + fmt.Sprintf(ErrHTTPResponseCodeError, response.StatusCode, c.Code)
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	answer, _ := regexp.Match(c.Answer, buf.Bytes())
	// check answer_present condition
	answerGood := (answer == (c.AnswerPresent != "absent")) && checkCode
	//logger.Printf("Answer: %v, AnswerPresent: %v, AnswerGood: %v", answer, c.AnswerPresent, answerGood)

	if !answerGood {
		answer := buf.String()
		if len(buf.String()) > 350 {
			answer = "Answer is too long, check the logs"
		}
		errorMessage := c.ErrorHeader + fmt.Sprintf(ErrHTTPAnswerTextError, answer, c.Answer, c.AnswerPresent)
		return time.Now().Sub(start), errors.New(errorMessage)
	}

	return time.Now().Sub(start), nil
}
