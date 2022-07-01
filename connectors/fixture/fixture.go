// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/log"
)

func TestSetup(funcs []func(string) func(), pkg string) func() {
	wg := new(sync.WaitGroup)
	var cleanUpFuncs []func()
	for _, f := range funcs {
		wg.Add(1)
		go func(fun func(string) func()) {
			defer wg.Done()
			cleanUpFuncs = append(cleanUpFuncs, fun(pkg))
		}(f)
	}
	wg.Wait()

	return func() {
		errs := cleanUp(cleanUpFuncs)
		log.Panic(errs, fixtureCleanUpError(pkg))
	}
}

func cleanUp(cleanUpFuncs []func()) error {
	wg := new(sync.WaitGroup)
	errs := make([]error, 0, len(cleanUpFuncs))
	for _, f := range cleanUpFuncs {
		wg.Add(1)
		go func(fun func()) {
			defer wg.Done()
			if err := recover(); err != nil {
				errs = append(errs, err.(error))
			}
			fun()
		}(f)
	}
	wg.Wait()
	if len(errs) > 1 {
		return multierror.Append(nil, errs...)
	} else if len(errs) == 1 {
		return errors.Wrapf(errs[0], "failed to cleanup all fixtures")
	}

	return nil
}

func fixtureCleanUpError(pkg string) error {
	return fmt.Errorf("%s %w", pkg, errFixtureCleanUp)
}
