package log

import (
	"github.com/sirupsen/logrus"
)

type TLogAlerter struct {
	Log *logrus.Logger
}

type TAlertDetails struct {
	Severity    string
	Message     string
	UUID        string
	ProjectName string
}
