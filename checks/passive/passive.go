package passive

import (
	"fmt"
	"my/checker/store"
	"time"
)

func (c TPassiveCheck) RealExecute() (time.Duration, error) {
	var (
		errorHeader, errorMessage string
	)

	if c.Timeout == "" {
		errorMessage = errorHeader + fmt.Sprintf(ErrEmptyTimeout)
		return 0, fmt.Errorf(errorMessage)
	}
	timeout, err := time.ParseDuration(c.Timeout)
	if err != nil {
		errorMessage = errorHeader + fmt.Sprintf(ErrTimeoutParseError, err)
		return 0, fmt.Errorf(errorMessage)
	}

	errorHeader = fmt.Sprintf(ErrPassiveError)

	check, err := store.Store.GetObjectByUUid(c.UUid)
	if err != nil {
		errorMessage = errorHeader + fmt.Sprintf(ErrCheckNotFound, err)
		return 0, fmt.Errorf(errorMessage)
	}

	dif := time.Now().Sub(check.LastPing)
	if dif > timeout {
		errorMessage = errorHeader + fmt.Sprintf(ErrCheckExpired, dif.String(), check.LastPing.String())
		return 0, fmt.Errorf(errorMessage)
	}

	return 0, nil
}
