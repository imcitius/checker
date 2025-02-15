package tcp

import (
	"my/checker/models"
)

func New(checkConfig models.TCheckConfig) ITCPCheck {
	var (
		realCheck TTCPCheck
	)

	if checkConfig.Type != "tcp" {
		logger.Errorf(ErrWrongCheckType, checkConfig.Type)
		return nil
	}

	if checkConfig.Host == "" {
		logger.Error(ErrEmptyHost)
		return nil
	}

	if checkConfig.Port == 0 {
		logger.Error(ErrEmptyPort)
		return nil
	}

	realCheck = TTCPCheck{
		Project:   checkConfig.Project,
		CheckName: checkConfig.Name,

		Host:    checkConfig.Host,
		Port:    checkConfig.Port,
		Count:   checkConfig.Count,
		Timeout: checkConfig.Timeout,
	}

	if checkConfig.Count == 0 {
		realCheck.Count = 1
	}

	return realCheck
}
