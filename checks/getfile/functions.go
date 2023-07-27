package getfile

import (
	"my/checker/config"
)

func New(checkConfig config.TCheckConfig) IGetFileCheck {
	var (
		realCheck TGetFileCheck
	)

	if checkConfig.Type != "getfile" {
		logger.Errorf(ErrWrongCheckType, checkConfig.Type)
		return nil
	}

	if checkConfig.Url == "" {
		logger.Error(ErrEmptyUrl)
		return nil
	}

	// allow to just download file, without check its hash
	// but if hash supplied, it must be valid
	if checkConfig.Hash == "d41d8cd98f00b204e9800998ecf8427e" {
		logger.Error(ErrEmptyHash)
		return nil
	}

	realCheck = TGetFileCheck{
		Project:   checkConfig.Project,
		CheckName: checkConfig.Name,

		Url:  checkConfig.Url,
		Hash: checkConfig.Hash,
		Size: checkConfig.Size,
	}

	return realCheck
}
