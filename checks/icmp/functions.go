package icmp

import (
	"my/checker/config"
)

func New(checkConfig config.TCheckConfig) IICMPCheck {
	var (
		realCheck TICMPCheck
	)

	if checkConfig.Type != "icmp" {
		logger.Error(ErrWrongCheckType, checkConfig.Type)
		return nil
	}

	if checkConfig.Host == "" {
		logger.Error(ErrEmptyHost)
		return nil
	}

	realCheck = TICMPCheck{
		Project:   checkConfig.Project,
		CheckName: checkConfig.Name,

		Host:    checkConfig.Host,
		Count:   checkConfig.Count,
		Timeout: checkConfig.Timeout,
	}

	if realCheck.Timeout == "" {
		realCheck.Timeout = configurer.Defaults.DefaultCheckParameters.Timeout
	}

	if realCheck.Count == 0 {
		realCheck.Count = 3
	}

	return &realCheck
}
