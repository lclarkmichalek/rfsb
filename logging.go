package rfsb

import "github.com/sirupsen/logrus"

// SetupDefaultLogging sets up a sane logrus logging config. Feel free not to use this; it's just for convenience
func SetupDefaultLogging() {
	logrus.SetLevel(logrus.InfoLevel)
}
