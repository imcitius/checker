package actors

import (
	"github.com/sirupsen/logrus"
)

type LogActor struct {
	Message string
}

func (l *LogActor) Act(msg string) error {
	logrus.Info(msg)
	return nil
}
