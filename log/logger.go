// SPDX-License-Identifier: BUSL-1.1

package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"

	"github.com/ice-blockchain/wintr/config"
)

// nolint:gochecknoinits // log is global, so it's initialization can be done in init
func init() {
	var appCfg cfg
	config.MustLoadFromKey("logger", &appCfg)

	var isJSON bool
	if strings.EqualFold(appCfg.Encoder, "json") {
		isJSON = true
	}
	setupLogger(isJSON, appCfg.Level)
	setupStdLibLogger(isJSON, appCfg.Level)
}

func setupLogger(isJSON bool, level string) {
	zerolog.DisableSampling(true)
	zerolog.ErrorStackMarshaler = errorStackMarshaller
	zerolog.InterfaceMarshalFunc = json.Marshal
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.DurationFieldUnit = time.Nanosecond
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}

	var err error
	logger, err = buildLogger(isJSON, level)
	if err != nil {
		panic(errors.Wrap(err, "failed to build setup logger"))
	}
}

func setupStdLibLogger(isJSON bool, level string) {
	l, err := buildLogger(isJSON, level)
	if err != nil {
		panic(errors.Wrap(err, "failed to build setup std lib logger"))
	}
	log.SetFlags(0)
	log.SetOutput(l)
}

func buildLogger(isJSON bool, level string) (*zerolog.Logger, error) { //nolint:revive // Control coupling is intended here.
	var logWriter io.Writer = os.Stderr
	if !isJSON {
		logWriter = &zerolog.ConsoleWriter{
			Out:        logWriter,
			TimeFormat: time.RFC3339Nano,
			PartsOrder: []string{
				zerolog.LevelFieldName,
				zerolog.TimestampFieldName,
				zerolog.MessageFieldName,
			},
			PartsExclude: []string{
				zerolog.ErrorFieldName,
				zerolog.ErrorStackFieldName,
				zerolog.CallerFieldName,
			},
		}
	}
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		return nil, errors.Wrap(err, "invalid logger level")
	}
	lgr := zerolog.New(logWriter).With().Timestamp().Stack().Logger().Level(lvl)

	return &lgr, nil
}

func errorStackMarshaller(err error) interface{} {
	m := pkgerrors.MarshalStack(err)
	if m == nil {
		return nil
	}
	frames, ok := m.([]map[string]string)
	if !ok || len(frames) == stackFramesToSkip {
		return nil
	}
	stacks := make([]string, 0, len(frames)-stackFramesToSkip)
	for _, frame := range frames[:len(frames)-stackFramesToSkip] {
		stacks = append(stacks, fmt.Sprintf("%s:%s:%s",
			frame[pkgerrors.StackSourceFileName],
			frame[pkgerrors.StackSourceLineName],
			frame[pkgerrors.StackSourceFunctionName]))
	}

	return strings.Join(stacks, "<<")
}
