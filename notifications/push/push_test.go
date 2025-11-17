// SPDX-License-Identifier: ice License 1.0

package push

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	stdlibtime "time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ice-blockchain/wintr/log"
)

const (
	testApplicationYAMLKeyBuffered    = "buffered"
	testApplicationYAMLKeyNonBuffered = "self"
)

const (
	testToken = "bogusToken" //nolint:gosec // Not a teal token.
	testTitle = "ice.io Test simple notification"
	testBody  = "This is a ice.io simple push notification from wintr/notifications/push tests "
)

func TestNonBufferedClientSend(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*stdlibtime.Second)
	defer cancel()
	isolatedClient := New(testApplicationYAMLKeyNonBuffered)
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
	responder := make(chan error)
	isolatedClient.Send(ctx, n1, responder)
	require.ErrorIs(t, <-responder, ErrInvalidDeviceToken)
	close(responder)
}

func TestBufferedClientSend(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*stdlibtime.Second)
	defer cancel()
	isolatedClient := New(testApplicationYAMLKeyBuffered)
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
	responder := make(chan error)
	isolatedClient.Send(ctx, n1, responder)
	require.ErrorIs(t, <-responder, ErrInvalidDeviceToken)
	close(responder)
}

func TestClientSendClosedResponder(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*stdlibtime.Second)
	defer cancel()
	isolatedClient := New(testApplicationYAMLKeyBuffered)
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
	responder := make(chan error)
	close(responder)
	isolatedClient.Send(ctx, n1, responder)
	stdlibtime.Sleep(fcmSendAllSlowProcessingMonitoringTickerDeadline + 5*stdlibtime.Second)
	<-ctx.Done()
}

func TestClientSendNoResponder(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*stdlibtime.Second)
	defer cancel()
	isolatedClient := New(testApplicationYAMLKeyBuffered)
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
	isolatedClient.Send(ctx, n1, nil)
	stdlibtime.Sleep(fcmSendAllSlowProcessingMonitoringTickerDeadline + 5*stdlibtime.Second)
	<-ctx.Done()
}

func TestClientSend_Buffering(t *testing.T) { //nolint:funlen // .
	t.Parallel()
	isolatedClient := New(testApplicationYAMLKeyBuffered)
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
	for range concurrency {
		go func() {
			defer wg.Done()
			stdlibtime.Sleep(stdlibtime.Duration(rand.Intn(4)) * stdlibtime.Second) //nolint:gosec // Good enough here.
			innerResponder := make(chan error)
			isolatedClient.Send(t.Context(), n1, innerResponder)
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
	isolatedClient := New(testApplicationYAMLKeyBuffered)
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
	for iter := range concurrency {
		go func() {
			defer wg.Done()
			stdlibtime.Sleep(stdlibtime.Duration(rand.Intn(4)) * stdlibtime.Second) //nolint:gosec // Good enough here.
			innerResponder := make(chan error)
			isolatedClient.Send(t.Context(), n1, innerResponder)
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

func TestNonBufferedClientBroadcast(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*stdlibtime.Second)
	defer cancel()
	isolatedClient := New(testApplicationYAMLKeyNonBuffered)
	defer func() {
		log.Panic(isolatedClient.Close())
	}()

	n1 := &Notification[SubscriptionTopic]{
		Data:     map[string]string{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Target:   "testing",
		Title:    "ice.io Test Broadcast",
		Body:     "This is a ice.io broadcast-ed notification from wintr/notifications/push tests " + uuid.NewString(),
		ImageURL: "https://miro.medium.com/max/1400/0*S1zFXEm7Cr9cdoKk",
	}
	require.NoError(t, isolatedClient.Broadcast(ctx, n1))
}

func TestBufferedClientBroadcast(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 30*stdlibtime.Second)
	defer cancel()
	isolatedClient := New(testApplicationYAMLKeyBuffered)
	defer func() {
		log.Panic(isolatedClient.Close())
	}()

	n1 := &Notification[SubscriptionTopic]{
		Data:     map[string]string{"deeplink": fmt.Sprintf("ice.app/something/%v", uuid.NewString())},
		Target:   "testing",
		Title:    "ice.io Test Broadcast",
		Body:     "This is a ice.io broadcast-ed notification from wintr/notifications/push tests " + uuid.NewString(),
		ImageURL: "https://miro.medium.com/max/1400/0*S1zFXEm7Cr9cdoKk",
	}
	require.NoError(t, isolatedClient.Broadcast(ctx, n1))
}

func BenchmarkBufferedClientSend(b *testing.B) {
	isolatedClient := New(testApplicationYAMLKeyBuffered)
	defer func() {
		log.Panic(isolatedClient.Close())
	}()
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
			isolatedClient.Send(b.Context(), n1, responder)
			require.ErrorIs(b, <-responder, ErrInvalidDeviceToken)
		}
	})
}

func BenchmarkNonBufferedClientSend(b *testing.B) {
	isolatedClient := New(testApplicationYAMLKeyNonBuffered)
	defer func() {
		log.Panic(isolatedClient.Close())
	}()
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
			isolatedClient.Send(b.Context(), n1, responder)
			require.ErrorIs(b, <-responder, ErrInvalidDeviceToken)
		}
	})
}
