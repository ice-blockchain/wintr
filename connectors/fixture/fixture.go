// SPDX-License-Identifier: BUSL-1.1

package fixture

import (
	"context"
	"flag"
	"os"
	"sort"
	"sync"
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
		connectorLifecycleHooks = new(ConnectorLifecycleHooks)
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

//nolint:funlen,gocognit,gocyclo // Alot of panic recovery. Mega ugly, but :shrug:, what can you do?
func (tr *testRunner) StartConnectorsIndefinitely(quit chan os.Signal) {
	//nolint:revive,staticcheck // String is good enough for local env.
	ctx := context.WithValue(context.Background(), applicationYAMLKeyContextValueKey, tr.applicationYAMLKey)
	beforeConnectorsStartedCleanUp := tr.BeforeConnectorsStarted(ctx)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(e.(error), "recovered panic"))
		}
		if err := beforeConnectorsStartedCleanUp(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanUp beforeConnectorsStarted"))
		}
	}()
	cleanUP := tr.startConnectors(ctx)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(e.(error), "recovered panic"))
		}
		defer func() {
			if e := recover(); e != nil {
				log.Error(errors.Wrapf(e.(error), "recovered panic"))
				if err := cleanUP(ctx); err != nil {
					log.Error(errors.Wrapf(err, "failed to cleanup connectors"))
				}
			}
		}()
		beforeConnectorsStoppedCleanUp := tr.BeforeConnectorsStopped(ctx)
		defer func() {
			if e := recover(); e != nil {
				log.Error(errors.Wrapf(e.(error), "recovered panic"))
			}
			if err := beforeConnectorsStoppedCleanUp(ctx); err != nil {
				log.Error(errors.Wrapf(err, "failed to cleanUp beforeConnectorsStopped"))
			}
		}()
		if err := cleanUP(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanup connectors"))
		}
		afterConnectorsStoppedCleanUp := tr.AfterConnectorsStopped(ctx)
		defer func() {
			if e := recover(); e != nil {
				log.Error(errors.Wrapf(e.(error), "recovered panic"))
			}
			if err := afterConnectorsStoppedCleanUp(ctx); err != nil {
				log.Error(errors.Wrapf(err, "failed to cleanUp afterConnectorsStopped"))
			}
		}()
	}()
	afterConnectorsStartedCleanUp := tr.AfterConnectorsStarted(ctx)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(e.(error), "recovered panic"))
		}
		if err := afterConnectorsStartedCleanUp(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanUp afterConnectorsStarted"))
		}
	}()
	log.Info("started connectors indefinitely")
	defer log.Info("stopping connectors...")

	<-quit
}

//nolint:funlen,gocognit,gocyclo // Alot of panic recovery. Mega ugly, but :shrug:, what can you do?
func (tr *testRunner) RunTests(m *testing.M) (code int) {
	flag.Parse()
	if testing.Short() {
		return m.Run()
	}
	//nolint:revive,staticcheck // String is good enough for tests.
	value := context.WithValue(context.Background(), applicationYAMLKeyContextValueKey, tr.applicationYAMLKey)
	//nolint:gomnd // It's not a magic number, it's the context deadline.
	ctx, cancel := context.WithTimeout(value, 30*time.Minute)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(e.(error), "recovered panic"))
			code = 1
		}
		cancel()
	}()
	beforeConnectorsStartedCleanUp := tr.BeforeConnectorsStarted(ctx)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(e.(error), "recovered panic"))
			code = 1
		}
		if err := beforeConnectorsStartedCleanUp(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanUp beforeConnectorsStarted"))
			code = 1
		}
	}()
	cleanUP := tr.startConnectors(ctx)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(e.(error), "recovered panic"))
			code = 1
		}
		defer func() {
			if e := recover(); e != nil {
				log.Error(errors.Wrapf(e.(error), "recovered panic"))
				code = 1
				if err := cleanUP(ctx); err != nil {
					log.Error(errors.Wrapf(err, "failed to cleanup connectors"))
				}
			}
		}()
		beforeConnectorsStoppedCleanUp := tr.BeforeConnectorsStopped(ctx)
		defer func() {
			if e := recover(); e != nil {
				log.Error(errors.Wrapf(e.(error), "recovered panic"))
				code = 1
			}
			if err := beforeConnectorsStoppedCleanUp(ctx); err != nil {
				log.Error(errors.Wrapf(err, "failed to cleanUp beforeConnectorsStopped"))
				code = 1
			}
		}()
		if err := cleanUP(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanup connectors"))
			code = 1
		}
		afterConnectorsStoppedCleanUp := tr.AfterConnectorsStopped(ctx)
		defer func() {
			if e := recover(); e != nil {
				log.Error(errors.Wrapf(e.(error), "recovered panic"))
				code = 1
			}
			if err := afterConnectorsStoppedCleanUp(ctx); err != nil {
				log.Error(errors.Wrapf(err, "failed to cleanUp afterConnectorsStopped"))
				code = 1
			}
		}()
	}()
	afterConnectorsStartedCleanUp := tr.AfterConnectorsStarted(ctx)
	defer func() {
		if e := recover(); e != nil {
			log.Error(errors.Wrapf(e.(error), "recovered panic"))
			code = 1
		}
		if err := afterConnectorsStartedCleanUp(ctx); err != nil {
			log.Error(errors.Wrapf(err, "failed to cleanUp afterConnectorsStarted"))
			code = 1
		}
	}()

	return m.Run()
}

func (tr *testRunner) startConnectors(ctx context.Context) ContextErrClose {
	cleanUpChan := make(chan orderedCleanUp, tr.testConnectorCount)
	for order := range tr.orderedSequence {
		connectors := tr.orderedTestConnectors[order]
		wg := new(sync.WaitGroup)
		wg.Add(len(connectors))
		for _, tc := range connectors {
			go func(tc TestConnector) {
				defer wg.Done()
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

	return func(cleanUpCtx context.Context) error {
		return errors.Wrapf(tr.cleanUpConnectors(cleanUpCtx), "`%v` fixture cleanup failed", tr.applicationYAMLKey)
	}
}

func (tr *testRunner) cleanUpConnectors(ctx context.Context) error {
	errChan := make(chan error, tr.testConnectorCount)
	for i := len(tr.orderedSequence) - 1; i >= 0; i-- {
		cleanUps := tr.orderedConnectorCleanUps[tr.orderedSequence[i]]
		wg := new(sync.WaitGroup)
		wg.Add(len(cleanUps))
		for _, f := range cleanUps {
			go func(cleanUp func(context.Context) error) {
				defer wg.Done()
				defer func() {
					if e := recover(); e != nil {
						errChan <- errors.Wrap(e.(error), "recovered from panic while cleaning up a connector")
					}
				}()
				errChan <- errors.Wrap(cleanUp(ctx), "failed to cleanup connector")
			}(f)
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
