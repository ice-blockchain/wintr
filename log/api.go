// SPDX-License-Identifier: BUSL-1.1

package log

import (
	"github.com/pkg/errors"
)

func Error(err error, fields ...interface{}) {
	if err == nil {
		return
	}
	errorEvent := logger.Err(err)
	if len(fields) > 0 {
		errorEvent = errorEvent.Fields(fields)
	}

	errorEvent.Send()
}

func Debug(msg string, fields ...interface{}) {
	debugEvent := logger.Debug()
	if len(fields) > 0 {
		debugEvent = debugEvent.Fields(fields)
	}

	debugEvent.Msg(msg)
}

func Info(msg string, fields ...interface{}) {
	infoEvent := logger.Info()
	if len(fields) > 0 {
		infoEvent = infoEvent.Fields(fields)
	}

	infoEvent.Msg(msg)
}

func Warn(msg string, fields ...interface{}) {
	warningEvent := logger.Warn()
	if len(fields) > 0 {
		warningEvent = warningEvent.Fields(fields)
	}

	warningEvent.Msg(msg)
}

func Fatal(anything any, fields ...interface{}) {
	if anything == nil {
		return
	}
	fatalEvent := logger.Fatal()
	if len(fields) > 0 {
		fatalEvent = fatalEvent.Fields(fields)
	}

	switch obj := anything.(type) {
	case error:
		fatalEvent.Err(obj).Send()

		return
	case string:
		fatalEvent.Msg(obj)

		return
	default:
		fatalEvent.Send()

		return
	}
}

func Panic(anything any, fields ...interface{}) {
	if anything == nil {
		return
	}
	panicEvent := logger.Panic()
	if len(fields) > 0 {
		panicEvent = panicEvent.Fields(fields)
	}

	switch obj := anything.(type) {
	case error:
		panicEvent.Err(obj).Send()

		return
	case string:
		panicEvent.Err(errors.New(obj)).Send()

		return
	default:
		panicEvent.Err(errors.Errorf("%#v", obj)).Send()

		return
	}
}

func Level() string {
	return logger.GetLevel().String()
}
