// SPDX-License-Identifier: ice License 1.0

package push

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"testing"
	stdlibtime "time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/log"
)

const (
	testApplicationYAMLKey = "self"
)

// .
var (
	//nolint:gochecknoglobals // It's a stateless singleton for tests.
	client Client
)

const (
	testToken = "bogusToken" //nolint:gosec // Not a teal token.
	testTitle = "ice.io Test simple notification"
	testBody  = "This is a ice.io simple push notification from wintr/notifications/push tests "
)

func TestMain(m *testing.M) {
	client = New(testApplicationYAMLKey)
	os.Exit(m.Run())
}

func TestClientSend(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	n1 := &Notification[DeviceToken]{
		Data:     map[string]string{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Target:   DeviceToken(testToken + uuid.NewString()),
		Title:    testTitle,
		Body:     testBody + uuid.NewString(),
		ImageURL: "https://miro.medium.com/max/1400/0*S1zFXEm7Cr9cdoKk",
	}
	responder := make(chan error)
	client.Send(ctx, n1, responder)
	require.ErrorIs(t, <-responder, ErrInvalidDeviceToken)
}

func TestClientSendClosedResponder(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	n1 := &Notification[DeviceToken]{
		Data:     map[string]string{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Target:   DeviceToken(testToken + uuid.NewString()),
		Title:    testTitle,
		Body:     testBody + uuid.NewString(),
		ImageURL: "https://miro.medium.com/max/1400/0*S1zFXEm7Cr9cdoKk",
	}
	responder := make(chan error)
	close(responder)
	client.Send(ctx, n1, responder)
	stdlibtime.Sleep(fcmSendAllSlowProcessingMonitoringTickerDeadline + 5*stdlibtime.Second)
	<-ctx.Done()
}

func TestClientSendNoResponder(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()
	n1 := &Notification[DeviceToken]{
		Data:     map[string]string{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Target:   DeviceToken(testToken + uuid.NewString()),
		Title:    testTitle,
		Body:     testBody + uuid.NewString(),
		ImageURL: "https://miro.medium.com/max/1400/0*S1zFXEm7Cr9cdoKk",
	}
	client.Send(ctx, n1, nil)
	stdlibtime.Sleep(fcmSendAllSlowProcessingMonitoringTickerDeadline + 5*stdlibtime.Second)
	<-ctx.Done()
}

func TestClientSend_Buffering(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	isolatedClient := New(testApplicationYAMLKey)
	defer func() {
		log.Panic(isolatedClient.Close())
	}()
	n1 := &Notification[DeviceToken]{
		Data:     map[string]string{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Target:   DeviceToken(testToken + uuid.NewString()),
		Title:    testTitle,
		Body:     testBody + uuid.NewString(),
		ImageURL: "https://miro.medium.com/max/1400/0*S1zFXEm7Cr9cdoKk",
	}
	wg := new(sync.WaitGroup)
	const concurrency = 100_000
	wg.Add(concurrency)
	responder := make(chan error, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			stdlibtime.Sleep(stdlibtime.Duration(rand.Intn(4)) * stdlibtime.Second) //nolint:gosec // Good enough here.
			innerResponder := make(chan error)
			isolatedClient.Send(context.Background(), n1, innerResponder)
			responder <- <-innerResponder
			close(innerResponder)
		}()
	}
	wg.Wait()
	close(responder)
	errCount := 0
	for err := range responder {
		require.ErrorIs(t, err, ErrInvalidDeviceToken)
		errCount++
	}
	require.Equal(t, concurrency, errCount)
}

func TestClientSend_Stability(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	isolatedClient := New(testApplicationYAMLKey)
	n1 := &Notification[DeviceToken]{
		Data:     map[string]string{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Target:   DeviceToken(testToken + uuid.NewString()),
		Title:    testTitle,
		Body:     testBody + uuid.NewString(),
		ImageURL: "https://miro.medium.com/max/1400/0*S1zFXEm7Cr9cdoKk",
	}
	wg := new(sync.WaitGroup)
	const concurrency = 100_000
	wg.Add(concurrency)
	responder := make(chan error, concurrency)
	for iter := 0; iter < concurrency; iter++ {
		go func() {
			defer wg.Done()
			stdlibtime.Sleep(stdlibtime.Duration(rand.Intn(4)) * stdlibtime.Second) //nolint:gosec // Good enough here.
			innerResponder := make(chan error)
			isolatedClient.Send(context.Background(), n1, innerResponder)
			responder <- <-innerResponder
			close(innerResponder)
		}()
		if iter == concurrency-(concurrency/10) {
			go func() {
				stdlibtime.Sleep(4 * stdlibtime.Second)
				log.Panic(isolatedClient.Close())
			}()
		}
	}
	wg.Wait()
	close(responder)
	errCount := 0
	for err := range responder {
		require.ErrorIs(t, err, ErrInvalidDeviceToken)
		errCount++
	}
	require.Equal(t, concurrency, errCount)
}

func TestClientBroadcast(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 30*stdlibtime.Second)
	defer cancel()

	n1 := &Notification[SubscriptionTopic]{
		Data:     map[string]string{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Target:   "testing",
		Title:    "ice.io Test Broadcast",
		Body:     "This is a ice.io broadcast-ed notification from wintr/notifications/push tests " + uuid.NewString(),
		ImageURL: "https://miro.medium.com/max/1400/0*S1zFXEm7Cr9cdoKk",
	}
	require.NoError(t, client.Broadcast(ctx, n1))
}

func BenchmarkClientSend(b *testing.B) {
	b.SetParallelism(1000)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n1 := &Notification[DeviceToken]{
				Data:     map[string]string{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
				Target:   DeviceToken(testToken + uuid.NewString()),
				Title:    testTitle,
				Body:     "This is a ice.io simple push notification from wintr/notifications/push benchmarks " + uuid.NewString(),
				ImageURL: "https://miro.medium.com/max/1400/0*S1zFXEm7Cr9cdoKk",
			}
			responder := make(chan error)
			client.Send(context.Background(), n1, responder)
			require.ErrorIs(b, <-responder, ErrInvalidDeviceToken)
		}
	})
}
