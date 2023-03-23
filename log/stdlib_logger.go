// SPDX-License-Identifier: ice License 1.0
//go:build !zerolog

package log

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/config"
)

const (
	debug = "debug"
	info  = "info"
)

// .
var (
	//nolint:gochecknoglobals // Immutable singleton.
	appCfg cfg
)

//nolint:gochecknoinits // log is global, so it's initialization can be done in init
func init() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix | log.LUTC | log.Llongfile | log.Lmicroseconds)
	config.MustLoadFromKey("logger", &appCfg)
}

func Error(err error, fields ...any) {
	if err == nil {
		return
	}
	vars := make([]string, 0, len(fields))
	for i := 0; i <= len(fields); i++ {
		vars = append(vars, "%v")
	}

	vals := make([]any, 0, len(fields)+1)
	vals = append(vals, err.Error())
	vals = append(vals, fields...)

	log.Printf(fmt.Sprintf("ERROR:%v", strings.Join(vars, " ")), vals...)
}

func Debug(msg string, fields ...any) {
	if !strings.EqualFold(appCfg.Level, debug) {
		return
	}
	vars := make([]string, 0, len(fields))
	for i := 0; i <= len(fields); i++ {
		vars = append(vars, "%v")
	}

	vals := make([]any, 0, len(fields)+1)
	vals = append(vals, msg)
	vals = append(vals, fields...)

	log.Printf(fmt.Sprintf("DEBUG:%v", strings.Join(vars, " ")), vals...)
}

func Info(msg string, fields ...any) {
	if strings.EqualFold(appCfg.Level, debug) {
		return
	}
	vars := make([]string, 0, len(fields))
	for i := 0; i <= len(fields); i++ {
		vars = append(vars, "%v")
	}

	vals := make([]any, 0, len(fields)+1)
	vals = append(vals, msg)
	vals = append(vals, fields...)

	log.Printf(fmt.Sprintf("INFO:%v", strings.Join(vars, " ")), vals...)
}

func Warn(msg string, fields ...any) {
	if lvl := strings.ToLower(appCfg.Level); lvl == debug || lvl == info {
		return
	}
	vars := make([]string, 0, len(fields))
	for i := 0; i <= len(fields); i++ {
		vars = append(vars, "%v")
	}

	vals := make([]any, 0, len(fields)+1)
	vals = append(vals, msg)
	vals = append(vals, fields...)

	log.Printf(fmt.Sprintf("WARN:%v", strings.Join(vars, " ")), vals...)
}

func Fatal(anything any, fields ...any) {
	if anything == nil {
		return
	}
	defer os.Exit(1)
	switch obj := anything.(type) {
	case error:
		Error(obj, fields...)

		return
	case string:
		Error(errors.New(obj), fields...)

		return
	default:
		Error(errors.Errorf("%#v", obj), fields...)

		return
	}
}

func Panic(anything any, fields ...any) {
	if anything == nil {
		return
	}
	defer func() {
		panic(anything)
	}()
	switch obj := anything.(type) {
	case error:
		Error(obj, fields...)

		return
	case string:
		Error(errors.New(obj), fields...)

		return
	default:
		Error(errors.Errorf("%#v", obj), fields...)

		return
	}
}

func Level() string {
	return appCfg.Level
}
