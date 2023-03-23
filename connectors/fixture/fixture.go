// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/log"
)

func NewTestRunner(applicationYAMLKey string, connectorLifecycleHooks *ConnectorLifecycleHooks, testConnectors ...TestConnector) TestRunner {
	orderedConnectors := make(map[int][]TestConnector, len(testConnectors))
	for _, tc := range testConnectors {
		orderedConnectors[tc.Order()] = append(orderedConnectors[tc.Order()], tc)
	}
	orderedSequence := make([]int, 0, len(orderedConnectors))
	for order := range orderedConnectors {
		orderedSequence = append(orderedSequence, order)
	}
	sort.Ints(orderedSequence)

	return &testRunner{
		applicationYAMLKey:      applicationYAMLKey,
		ConnectorLifecycleHooks: processHooks(connectorLifecycleHooks),
		orderedTestConnectors:   orderedConnectors,
		orderedSequence:         orderedSequence,
		testConnectorCount:      len(testConnectors),
	}
}

func processHooks(connectorLifecycleHooks *ConnectorLifecycleHooks) *ConnectorLifecycleHooks {
	if connectorLifecycleHooks == nil {
		connectorLifecycleHooks = new(ConnectorLifecycleHooks) //nolint:revive // That's intended.
	}
	defHook := func(context.Context) ContextErrClose {
		return func(context.Context) error {
			return nil
		}
	}
	if connectorLifecycleHooks.AfterConnectorsStarted == nil {
		connectorLifecycleHooks.AfterConnectorsStarted = defHook
	}
	if connectorLifecycleHooks.BeforeConnectorsStarted == nil {
		connectorLifecycleHooks.BeforeConnectorsStarted = defHook
	}
	if connectorLifecycleHooks.AfterConnectorsStopped == nil {
		connectorLifecycleHooks.AfterConnectorsStopped = defHook
	}
	if connectorLifecycleHooks.BeforeConnectorsStopped == nil {
		connectorLifecycleHooks.BeforeConnectorsStopped = defHook
	}

	return connectorLifecycleHooks
}

//nolint:funlen,gocognit,revive // Alot of panic recovery. Mega ugly, but :shrug:, what can you do?
func (tr *testRunner) StartConnectorsIndefinitely() {
	//nolint:revive,staticcheck // String is good enough for local env.
	ctx := context.WithValue(context.Background(), applicationYAMLKeyContextValueKey, tr.applicationYAMLKey)
	beforeConnectorsStartedCleanUp := tr.BeforeConnectorsStarted(ctx)
	defer func() {
		log.Error(errors.Wrapf(mapErr(recover()), "recovered panic"))
		if err := beforeConnectorsStartedCleanUp(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanUp beforeConnectorsStarted"))
		}
	}()
	cleanUP := tr.startConnectors(ctx)
	defer func() {
		log.Error(errors.Wrapf(mapErr(recover()), "recovered panic"))
		defer func() {
			if e := recover(); e != nil {
				log.Error(errors.Wrapf(mapErr(e), "recovered panic"))
				if err := cleanUP(ctx); err != nil {
					log.Error(errors.Wrapf(err, "failed to cleanup connectors"))
				}
			}
		}()
		beforeConnectorsStoppedCleanUp := tr.BeforeConnectorsStopped(ctx)
		defer func() {
			log.Error(errors.Wrapf(mapErr(recover()), "recovered panic"))
			if err := beforeConnectorsStoppedCleanUp(ctx); err != nil {
				log.Error(errors.Wrapf(err, "failed to cleanUp beforeConnectorsStopped"))
			}
		}()
		if err := cleanUP(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanup connectors"))
		}
		afterConnectorsStoppedCleanUp := tr.AfterConnectorsStopped(ctx)
		defer func() {
			log.Error(errors.Wrapf(mapErr(recover()), "recovered panic"))
			if err := afterConnectorsStoppedCleanUp(ctx); err != nil {
				log.Error(errors.Wrapf(err, "failed to cleanUp afterConnectorsStopped"))
			}
		}()
	}()
	afterConnectorsStartedCleanUp := tr.AfterConnectorsStarted(ctx)
	defer func() {
		log.Error(errors.Wrapf(mapErr(recover()), "recovered panic"))
		if err := afterConnectorsStartedCleanUp(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanUp afterConnectorsStarted"))
		}
	}()
	log.Info("started connectors indefinitely")
	defer log.Info("stopping connectors...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

//nolint:funlen,gocognit,gocyclo,cyclop,revive // Alot of panic recovery. Mega ugly, but :shrug:, what can you do?
func (tr *testRunner) RunTests(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}
	var exitCode int
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(mapErr(e), "recovered panic"))
			exitCode = 1
		}
		os.Exit(exitCode)
	}()
	//nolint:revive,staticcheck // String is good enough for tests.
	value := context.WithValue(context.Background(), applicationYAMLKeyContextValueKey, tr.applicationYAMLKey)
	//nolint:gomnd // It's not a magic number, it's the context deadline.
	ctx, cancel := context.WithTimeout(value, 30*time.Minute)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(mapErr(e), "recovered panic"))
			exitCode = 1
		}
		cancel()
	}()
	beforeConnectorsStartedCleanUp := tr.BeforeConnectorsStarted(ctx)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(mapErr(e), "recovered panic"))
			exitCode = 1
		}
		if err := beforeConnectorsStartedCleanUp(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanUp beforeConnectorsStarted"))
			exitCode = 1
		}
	}()
	cleanUP := tr.startConnectors(ctx)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(mapErr(e), "recovered panic"))
			exitCode = 1
		}
		defer func() {
			if e := recover(); e != nil {
				log.Error(errors.Wrapf(mapErr(e), "recovered panic"))
				exitCode = 1
				if err := cleanUP(ctx); err != nil {
					log.Error(errors.Wrapf(err, "failed to cleanup connectors"))
				}
			}
		}()
		beforeConnectorsStoppedCleanUp := tr.BeforeConnectorsStopped(ctx)
		defer func() {
			if e := recover(); e != nil {
				log.Error(errors.Wrapf(mapErr(e), "recovered panic"))
				exitCode = 1
			}
			if err := beforeConnectorsStoppedCleanUp(ctx); err != nil {
				log.Error(errors.Wrapf(err, "failed to cleanUp beforeConnectorsStopped"))
				exitCode = 1
			}
		}()
		if err := cleanUP(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanup connectors"))
			exitCode = 1
		}
		afterConnectorsStoppedCleanUp := tr.AfterConnectorsStopped(ctx)
		defer func() {
			if e := recover(); e != nil {
				log.Error(errors.Wrapf(mapErr(e), "recovered panic"))
				exitCode = 1
			}
			if err := afterConnectorsStoppedCleanUp(ctx); err != nil {
				log.Error(errors.Wrapf(err, "failed to cleanUp afterConnectorsStopped"))
				exitCode = 1
			}
		}()
	}()
	afterConnectorsStartedCleanUp := tr.AfterConnectorsStarted(ctx)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(mapErr(e), "recovered panic"))
			exitCode = 1
		}
		if err := afterConnectorsStartedCleanUp(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanUp afterConnectorsStarted"))
			exitCode = 1
		}
	}()

	exitCode = m.Run()
}

//nolint:funlen,revive // Alot of error handling.
func (tr *testRunner) startConnectors(ctx context.Context) (cleanUpAll ContextErrClose) {
	cleanUpChan := make(chan orderedCleanUp, tr.testConnectorCount)
	var paniced uint64
	for order := range tr.orderedSequence {
		connectors := tr.orderedTestConnectors[order]
		wg := new(sync.WaitGroup)
		wg.Add(len(connectors))
		for _, tc := range connectors {
			go func(tc TestConnector) {
				defer wg.Done()
				defer func() {
					if e := recover(); e != nil {
						log.Error(errors.Wrapf(mapErr(e), "recovered from panic"))
						cleanUpChan <- orderedCleanUp{order: tc.Order(), cleanUp: func(context.Context) error { return nil }} //nolint:revive // Wrong
						atomic.AddUint64(&paniced, 1)
					}
				}()
				cleanUp := tc.Setup(ctx)
				cleanUpChan <- orderedCleanUp{order: tc.Order(), cleanUp: cleanUp}
			}(tc)
		}
		wg.Wait()
	}
	close(cleanUpChan)
	tr.orderedConnectorCleanUps = make(map[int][]func(context.Context) error, tr.testConnectorCount)
	for orderedConnectorCleanUp := range cleanUpChan {
		order := orderedConnectorCleanUp.order
		tr.orderedConnectorCleanUps[order] = append(tr.orderedConnectorCleanUps[order], orderedConnectorCleanUp.cleanUp)
	}
	cleanUpAll = func(cleanUpCtx context.Context) error {
		return errors.Wrapf(tr.cleanUpConnectors(cleanUpCtx), "`%v` fixture cleanup failed", tr.applicationYAMLKey)
	}
	if paniced > 0 {
		log.Error(errors.Wrapf(cleanUpAll(ctx), "failed to cleanup connectors due to premature panic recovery"))
		log.Panic(errors.Errorf("premature panic while starting connectors for %v", tr.applicationYAMLKey))
	}

	return cleanUpAll
}

func (tr *testRunner) cleanUpConnectors(ctx context.Context) error {
	errChan := make(chan error, tr.testConnectorCount)
	for i := len(tr.orderedSequence) - 1; i >= 0; i-- {
		cleanUps := tr.orderedConnectorCleanUps[tr.orderedSequence[i]]
		wg := new(sync.WaitGroup)
		wg.Add(len(cleanUps))
		for _, fn := range cleanUps {
			go func(cleanUp func(context.Context) error) {
				defer wg.Done()
				defer func() {
					if e := recover(); e != nil {
						errChan <- errors.Wrap(mapErr(e), "recovered from panic while cleaning up a connector")
					}
				}()
				errChan <- errors.Wrap(cleanUp(ctx), "failed to cleanup connector")
			}(fn)
		}
		wg.Wait()
	}
	close(errChan)
	errs := make([]error, 0, tr.testConnectorCount)
	for err := range errChan {
		errs = append(errs, err)
	}

	return errors.Wrap(multierror.Append(nil, errs...).ErrorOrNil(), "some cleanup logic failed")
}

func mapErr(maybeError any) error {
	if maybeError == nil {
		return nil
	}
	if errString, ok := maybeError.(string); ok {
		return errors.New(errString)
	}
	if actualErr, ok := maybeError.(error); ok {
		return actualErr
	}

	return errors.Errorf("unexpected error: %#v", maybeError)
}
