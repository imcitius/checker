package cmd

import (
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func Execute1() {

	Config.runScheduler()

}
