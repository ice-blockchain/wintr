// SPDX-License-Identifier: ice License 1.0

package config

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

//nolint:gochecknoinits // Because we load the configs once, for the whole runtime
func init() {
	loadFirstApplicationConfigFile()
	dotEnvPath := `.env`
	for range 5 {
		if err := godotenv.Load(dotEnvPath); err == nil {
			break
		}
		dotEnvPath = fmt.Sprintf(`../%v`, dotEnvPath)
	}
}

func MustLoadFromKey(key string, cfg any) {
	if err := viper.UnmarshalKey(key, cfg); err != nil {
		log.Panic(errors.Wrapf(err, "failed to load config by key %q", key))
	}
}

func loadFirstApplicationConfigFile() {
	for _, f := range findAllApplicationConfigFiles() {
		viper.SetConfigFile(f)
		if err := viper.ReadInConfig(); err == nil {
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			log.Panic(err)
		}
	}

	log.Panic(errors.New("could not find any application.yaml files"))
}

func findAllApplicationConfigFiles() []string {
	var files []string
	var hints []string

	if p, err := os.Getwd(); err == nil {
		hints = append(hints, p)
	}
	if p, err := os.Executable(); err == nil {
		hints = append(hints, path.Dir(filepath.Join(p, "..")))
	}

	for _, dir := range hints {
		pattern := filepath.Join(dir, ".testdata", "application.yaml")
		if f, err := filepath.Glob(pattern); err != nil {
			log.Println(errors.Wrapf(err, "glob failed for [%v]", pattern))
		} else {
			files = append(files, f...)
		}
		pattern = filepath.Join(dir, "application.yaml")
		if f, err := filepath.Glob(pattern); err != nil {
			log.Println(errors.Wrapf(err, "glob failed for [%v]", pattern))
		} else {
			files = append(files, f...)
		}
	}
	files = append(files, relativeFiles()...)

	return files
}

func relativeFiles() []string {
	var files []string
	//nolint:dogsled // Because those 3 blank identifiers are useless
	_, callerFile, _, _ := runtime.Caller(0)
	pattern := filepath.Join(filepath.Dir(callerFile), "..", "application.yaml")
	if f, err := filepath.Glob(pattern); err != nil {
		log.Println(errors.Wrapf(err, "glob failed for [%v]", pattern))
	} else {
		files = append(files, f...)
	}
	pattern = filepath.Join(filepath.Dir(callerFile), "..", "..", "application.yaml")
	if f, err := filepath.Glob(pattern); err != nil {
		log.Println(errors.Wrapf(err, "glob failed for [%v]", pattern))
	} else {
		files = append(files, f...)
	}

	return files
}
