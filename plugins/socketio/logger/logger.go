package logger

import "github.com/robertkonga/yekonga-server/helper/logger"

func Error(msg string, err error) {
	logger.Error(msg, "err", err.Error())
}

func Info(msg string, args ...interface{}) {
	logger.Info(msg, args)
}
