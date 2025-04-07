package logger

import (
	log "github.com/sirupsen/logrus"
)

func Init() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	log.SetLevel(log.DebugLevel)
}

func Info(args ...interface{}) {
	log.Info(args...)
}

func Error(args ...interface{}) {
	log.Error(args...)
}

func Debug(args ...interface{}) {
	log.Debug(args...)
}

func Warn(args ...interface{}) {
	log.Warn(args...)
}

func Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

func Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

func Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}
