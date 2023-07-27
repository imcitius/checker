package http

import (
	"crypto/tls"
	"fmt"
	"my/checker/config"
	"regexp"
	"time"
)

func New(checkConfig config.TCheckConfig) IHTTPCheck {
	var (
		realCheck THTTPCheck
	)

	if checkConfig.Type != "http" {
		logger.Error(ErrWrongCheckType, checkConfig.Type)
		return nil
	}

	realCheck = THTTPCheck{
		Project:             checkConfig.Project,
		CheckName:           checkConfig.Name,
		Auth:                checkConfig.Auth,
		Code:                checkConfig.Code,
		Answer:              checkConfig.Answer,
		AnswerPresent:       checkConfig.AnswerPresent,
		Url:                 checkConfig.Url,
		Timeout:             checkConfig.Timeout,
		StopFollowRedirects: checkConfig.StopFollowRedirects,
		Headers:             checkConfig.Headers,
		Cookies:             checkConfig.Cookies,
		SkipCheckSSL:        checkConfig.SkipCheckSSL,
		SSLExpirationPeriod: checkConfig.SSLExpirationPeriod,

		TlsConfig: tls.Config{
			InsecureSkipVerify: checkConfig.SkipCheckSSL,
		},
	}

	if realCheck.SSLExpirationPeriod == "" {
		realCheck.SSLExpirationPeriod = configurer.Defaults.DefaultCheckParameters.SSLExpirationPeriod
	}

	if realCheck.Timeout == "" {
		realCheck.Timeout = configurer.Defaults.DefaultCheckParameters.Timeout
	}

	pattern := regexp.MustCompile("(.*)://")
	realCheck.Scheme = pattern.FindStringSubmatch(realCheck.Url)[0]

	SSLExpirationPeriodParsed, err := time.ParseDuration(realCheck.SSLExpirationPeriod)
	if err != nil {
		err := fmt.Errorf(ErrParseSSlTimeout, err)
		logger.Errorf(err.Error())
		return nil
	}
	realCheck.SSLExpirationPeriodParsed = SSLExpirationPeriodParsed

	return &realCheck
}

func checkAnswerCode(codes []int, code int) bool {
	found := false
	// init answer codes slice if empty
	if len(codes) == 0 {
		codes = []int{200}
	}
	for _, c := range codes {
		if c == code {
			found = true
		}
	}
	return found
}
