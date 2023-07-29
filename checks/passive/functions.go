package passive

import (
	"my/checker/config"
)

func New(checkConfig config.TCheckConfig) IPassiveCheck {
	var (
		realCheck TPassiveCheck
	)

	if checkConfig.Type != "passive" {
		logger.Error(ErrWrongCheckType, checkConfig.Type)
		return nil
	}

	if checkConfig.Timeout == "" {
		logger.Error(ErrEmptyTimeout)
		return nil
	}

	realCheck = TPassiveCheck{
		Project:   checkConfig.Project,
		CheckName: checkConfig.Name,
		UUid:      checkConfig.UUid,

		Timeout: checkConfig.Timeout,
	}

	return realCheck
}
