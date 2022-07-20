package stdlogger

import (
	"github.com/sirupsen/logrus"
)

var Fields = logrus.Fields{}

func Trace(args ...interface{}) {
	logrus.WithFields(Fields).Debug(args...)
}

func Debug(args ...interface{}) {
	logrus.WithFields(Fields).Debug(args...)
}

func Info(args ...interface{}) {
	logrus.WithFields(Fields).Info(args...)
}

func Warn(args ...interface{}) {
	logrus.WithFields(Fields).Warn(args...)
}

func Error(args ...interface{}) {
	logrus.WithFields(Fields).Error(args...)
}

func Panic(args ...interface{}) {
	logrus.WithFields(Fields).Panic(args...)
}

func Fatal(args ...interface{}) {
	logrus.WithFields(Fields).Fatal(args...)
}

func Tracef(format string, args ...interface{}) {
	logrus.WithFields(Fields).Tracef(format, args...)
}

func Debugf(format string, args ...interface{}) {
	logrus.WithFields(Fields).Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	logrus.WithFields(Fields).Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	logrus.WithFields(Fields).Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	logrus.WithFields(Fields).Errorf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	logrus.WithFields(Fields).Panicf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	logrus.WithFields(Fields).Fatalf(format, args...)
}

func Exit(code int) {
	logrus.Exit(code)
}
