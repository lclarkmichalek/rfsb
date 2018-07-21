package rfsb

import "github.com/sirupsen/logrus"

func SetupDefaultLogging() {
	logrus.SetLevel(logrus.InfoLevel)
}
