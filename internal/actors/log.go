package actors

import (
	"github.com/sirupsen/logrus"
)

type LogActor struct {
	Message string
}

func (l *LogActor) Act() error {
	logrus.Info(l.Message)
	return nil
}
