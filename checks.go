package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/sparrc/go-ping"
)

type httpHeader map[string]string

// UniversalCheck - Interface to any possible healthcheck
type UniversalCheck interface {
	Execute() (result, error)
	UUID() string
	HostName() string
}

type result interface{}

type urlCheck struct {
	URL           string       `json:"url"`
	Code          uint         `json:"code"`
	Answer        string       `json:"answer"`
	AnswerPresent string       `json:"answer_present"`
	Headers       []httpHeader `json:"headers"`
	uuID          string
	Mode          string
}

type icmpPingCheck struct {
	Host    string
	Timeout time.Duration
	Count   uint
	uuID    string
	Mode    string
}

type tcpPingCheck struct {
	Host     string
	Timeout  time.Duration
	Port     uint
	Attempts uint
	uuID     string
	Mode     string
}

func (c urlCheck) UUID() string {
	var uuID string
	for _, project := range Config.Projects {
		for _, check := range project.Checks.URLChecks {
			if check.URL == c.URL {
				uuID = check.uuID
			}
		}
	}
	return uuID
}

func (c urlCheck) HostName() string {
	var host string
	for _, project := range Config.Projects {
		for _, check := range project.Checks.URLChecks {
			if check.URL == c.URL {
				host = check.URL
			}
		}
	}
	return host
}

func (c icmpPingCheck) UUID() string {
	var uuID string
	for _, project := range Config.Projects {
		for _, check := range project.Checks.TCPPingChecks {
			if check.Host == c.Host {
				uuID = check.uuID
			}
		}
	}
	return uuID
}

func (c icmpPingCheck) HostName() string {
	var host string
	for _, project := range Config.Projects {
		for _, check := range project.Checks.TCPPingChecks {
			if check.Host == c.Host {
				host = check.Host
			}
		}
	}
	return host
}

func (c tcpPingCheck) UUID() string {
	var uuID string
	for _, project := range Config.Projects {
		for _, check := range project.Checks.ICMPPingChecks {
			if check.Host == c.Host {
				uuID = check.uuID
			}
		}
	}
	return uuID
}

func (c tcpPingCheck) HostName() string {
	var host string
	for _, project := range Config.Projects {
		for _, check := range project.Checks.ICMPPingChecks {
			if check.Host == c.Host {
				host = check.Host
			}
		}
	}
	return host
}

func (c urlCheck) Execute() (result, error) {
	var (
		answerPresent bool = true
		checkNum      uint
		response      result
		checkerr      error
	)
	fmt.Println("test: ", c.URL)
	_, err := url.Parse(c.URL)
	if err != nil {
		log.Fatal(err)
	}
	checkNum++

	client := &http.Client{}
	req, err := http.NewRequest("GET", c.URL, nil)

	// if custom headers requested
	if c.Headers != nil {
		for _, headers := range c.Headers {
			for header, value := range headers {
				req.Header.Add(header, value)
			}
		}
	}
	// log.Printf("http request: %v", req)
	response, err = client.Do(req)
	if err != nil {
		errorMessage := fmt.Sprintf("Http asnwer error: %+v", err)
		return response, errors.New(errorMessage)
	}

	if response.(*http.Response).Body != nil {
		defer response.(*http.Response).Body.Close()
	} else {
		errorMessage := fmt.Sprintf("Http empty body: %+v", response.(*http.Response))
		return response, errors.New(errorMessage)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.(*http.Response).Body)

	// check that response code is correct
	code := c.Code == uint(response.(*http.Response).StatusCode)
	if !code {
		errorMessage := fmt.Sprintf("Http code error: %d (want %d)", response.(*http.Response).StatusCode, c.Code)
		return response, errors.New(errorMessage)
	}

	answer, _ := regexp.Match(c.Answer, buf.Bytes())
	// check answer_present condition
	if c.AnswerPresent == "absent" {
		answerPresent = false
	}
	answerGood := (answer == answerPresent) && code
	// log.Printf("Answer: %v, AnswerPresent: %v, AnswerGood: %v", answer, urlcheck.AnswerPresent, answerGood)

	if !answerGood {
		errorMessage := fmt.Sprintf("Http check error: %+v", response.(*http.Response))
		return response, errors.New(errorMessage)
	}
	return response, checkerr
}

func (c icmpPingCheck) Execute() (result, error) {
	var (
		checkNum uint
		checkerr error
	)

	fmt.Println("icmp ping test: ", c.Host)
	checkNum++
	pinger, _ := ping.NewPinger(c.Host)
	pinger.Count = int(c.Count)
	pinger.Timeout = c.Timeout * 1000 * 1000 //milliseconds
	pinger.Run()
	stats := pinger.Statistics()

	// log.Printf("Ping host %s, res: %+v (err: %+v, stats: %+v)", c.Host, pinger, err, stats)

	if stats.PacketLoss == 0 && stats.AvgRtt < c.Timeout {
		return stats, checkerr
	}
	return stats, checkerr
}

func (c tcpPingCheck) Execute() (result, error) {
	var (
		checkNum      uint
		checkAttempts uint
		checkerr      error
	)

	checkNum++
	checkhost := fmt.Sprintf("%s:%d", c.Host, c.Port)
	timeout := c.Timeout * 1000 * 1000 // millisecond
	fmt.Printf("tcp ping test: %s\n", checkhost)

	for checkAttempts < c.Attempts {
		startTime := time.Now()
		conn, err := net.DialTimeout("tcp", checkhost, timeout)
		endTime := time.Now()

		if err == nil {
			defer conn.Close()
			var t = float64(endTime.Sub(startTime)) / float64(time.Millisecond)
			log.Printf("Connection to host %v succeed, took %v millisec", conn.RemoteAddr().String(), t)
			return checkAttempts, err
		}

		log.Printf("connection failed: %v (attempt %d)\n", err, checkAttempts)
		checkAttempts++
	}
	return checkAttempts, checkerr
}
