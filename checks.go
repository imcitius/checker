package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/sparrc/go-ping"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

func (c *Check) Execute(p *Project) error {

	switch c.Type {
	case "http":
		//log.Printf("http check execute: %+v\n", c.Host)
		err := runHTTPCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
		return err
	case "icmp":
		//log.Printf("icmp check execute %+v\n", c)
		err := runICMPCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
		return err
	case "tcp":
		//log.Printf("tcp check execute %+v\n", c)
		err := runTCPCheck(c, p)
		if err == nil {
			return nil
		} else {
			c.LastResult = false
		}
		return err
	default:
		return errors.New("check not implemented")
	}
}

func (c *Check) UUID() string {
	return c.uuID
}

func (c *Check) HostName() string {
	return c.Host
}

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

	// log.Printf("http request: %v", req)
	response, err := client.Do(req)
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

func runICMPCheck(c *Check, p *Project) error {
	var (
		errorHeader, errorMessage string
	)

	errorHeader = fmt.Sprintf("ICMP error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.uuID)

	fmt.Println("icmp ping test: ", c.Host)
	pinger, err := ping.NewPinger(c.Host)
	pinger.Count = c.Count
	pinger.Timeout = c.Timeout * time.Millisecond
	pinger.Run()
	stats := pinger.Statistics()

	//log.Printf("Ping host %s, res: %+v (err: %+v, stats: %+v)", c.Host, pinger, err, stats)

	if err == nil && stats.PacketLoss == 0 {
		return nil
	} else {
		switch {
		case stats.PacketLoss > 0:
			//log.Printf("Ping stats: %+v", stats)
			errorMessage = errorHeader + fmt.Sprintf("ping error: %v percent packet loss\n", stats.PacketLoss)
		default:
			//log.Printf("Ping stats: %+v", stats)
			errorMessage = errorHeader + fmt.Sprintf("other ping error: %+v\n", err)
		}
	}

	//log.Println(errorMessage)
	return errors.New(errorMessage)

}

func runTCPCheck(c *Check, p *Project) error {
	var (
		errorHeader, errorMessage string
		checkAttempts             int
	)

	//log.Panic(projectName)

	errorHeader = fmt.Sprintf("TCP error at project: %s\nCheck Host: %s\nCheck UUID: %s\n", p.Name, c.Host, c.uuID)

	fmt.Println("tcp ping test: ", c.Host)

	timeout := c.Timeout * time.Millisecond

	for checkAttempts < c.Attempts {
		//startTime := time.Now()
		conn, err := net.DialTimeout("tcp", c.Host+":"+c.Port, timeout)
		//endTime := time.Now()

		if err == nil {
			conn.Close()
			//t := float64(endTime.Sub(startTime)) / float64(time.Millisecond)
			//log.Printf("Connection to host %v succeed, took %v millisec", conn.RemoteAddr().String(), t)
			return nil
		}

		errorMessage = errorHeader + fmt.Sprintf("connection to host %s failed: %v (attempt %d)\n", c.Host+":"+c.Port, err, checkAttempts)
		//log.Printf(errorMessage)
		checkAttempts++
	}

	fmt.Println(errorMessage)
	return errors.New(errorMessage)

}

func (c *Check) CeaseAlerts() error {
	log.Printf("Old mode: %s", c.Mode)
	c.Mode = "quiet"
	log.Printf("New mode: %s", c.Mode)
	return nil
}

func (c *Check) EnableAlerts() error {
	log.Printf("Old mode: %s", c.Mode)
	c.Mode = "loud"
	log.Printf("New mode: %s", c.Mode)
	return nil
}
