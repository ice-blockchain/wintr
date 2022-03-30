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

	"github.com/ICE-Blockchain/wintr/config"
)

// nolint:gochecknoinits // log is global, so it's initialization can be done in init
func init() {
	var c cfg
	config.MustLoadFromKey("logger", &c)

	var isJSON bool
	if strings.EqualFold(c.Encoder, "json") {
		isJSON = true
	}
	setupLogger(isJSON, c.Level)
	setupStdLibLogger(isJSON, c.Level)
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

func buildLogger(isJSON bool, level string) (*zerolog.Logger, error) {
	var w io.Writer = os.Stderr
	if !isJSON {
		w = &zerolog.ConsoleWriter{
			Out:        w,
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
	l := zerolog.New(w).With().Timestamp().Stack().Logger().Level(lvl)

	return &l, nil
}

func errorStackMarshaller(err error) interface{} {
	m := pkgerrors.MarshalStack(err)
	if m == nil {
		return nil
	}
	frames := m.([]map[string]string)
	if len(frames) == stackFramesToSkip {
		return nil
	}
	r := make([]string, 0, len(frames)-stackFramesToSkip)
	for _, frame := range frames[:len(frames)-stackFramesToSkip] {
		r = append(r, fmt.Sprintf("%s:%s:%s",
			frame[pkgerrors.StackSourceFileName],
			frame[pkgerrors.StackSourceLineName],
			frame[pkgerrors.StackSourceFunctionName]))
	}

	return strings.Join(r, "<<")
}
