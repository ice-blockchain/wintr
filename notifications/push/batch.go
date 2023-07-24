// SPDX-License-Identifier: ice License 1.0

package push

import (
	"context"
	"sync"

	fcm "firebase.google.com/go/v4/messaging"
	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/log"
)

func (s *pushNotificationsCollectingSink) newNotificationBatch(notifications ...*notification) *notificationBatch {
	return &notificationBatch{
		sink:                     s,
		notifications:            notifications,
		devicesWithInvalidTokens: make(map[int]struct{}, len(notifications)),
		devicesNotified:          make(map[int]struct{}, len(notifications)),
		devicesFailed:            make(map[int]error, len(notifications)),
	}
}

func (b *notificationBatch) process() {
	ctx, cancel := context.WithTimeout(context.Background(), requestDeadline)
	defer cancel()
	op := func() error {
		return b.fcmSendAll(ctx)
	}
	log.Error(errors.Wrapf(retry(ctx, op), "[%v] permanently failed to send some push notifications", b.sink.applicationYAMLKey))

	if len(b.notifications) != (len(b.devicesNotified) + len(b.devicesFailed)) {
		for i := range b.notifications {
			if _, notified := b.devicesNotified[i]; !notified {
				b.devicesFailed[i] = errors.New("sending push notification to device token was not attempted due to deadline exhaustion")
			}
		}
	}
	b.respond()
}

func (b *notificationBatch) respond() {
	wg := new(sync.WaitGroup)
	wg.Add(len(b.notifications))
	for idx := range b.notifications {
		go func(ix int) {
			responder := b.notifications[ix].responder
			notif := b.notifications[ix].notification
			err := b.devicesFailed[ix]
			defer wg.Done()
			defer func() {
				if recoveredErr := recover(); recoveredErr != nil {
					log.Error(mapErr(recoveredErr))
					log.Error(errors.Wrapf(err, "[panic recover] sending push notification to device token failed permanently: %#v", notif))
				}
			}()
			if responder != nil {
				responder <- errors.Wrapf(err, "sending push notification to device token failed permanently: %#v", notif)
			} else {
				log.Error(errors.Wrapf(err, "sending push notification to device token failed permanently: %#v", notif))
			}
		}(idx)
	}
	wg.Wait()
}

func (b *notificationBatch) fcmSendAll(ctx context.Context) error { //nolint:funlen // Because it would be worse to break it.
	messages, deviceIndices := buildFCMMessages(b.notifications, b.devicesWithInvalidTokens, b.devicesNotified)
	batchR, err := b.sink.client.SendAll(ctx, messages) //nolint:staticcheck // Will do that later.
	if err != nil {
		if errors.Is(err, ctx.Err()) {
			//nolint:wrapcheck // no need, its terminal and internal.
			return backoff.Permanent(ctx.Err())
		}

		//nolint:wrapcheck // no need, cuz its recursion.
		return err
	}
	for idx, resp := range batchR.Responses {
		if !resp.Success && resp.Error != nil {
			var rErr error
			if fcm.IsInvalidArgument(resp.Error) || fcm.IsUnregistered(resp.Error) || fcm.IsSenderIDMismatch(resp.Error) {
				b.devicesWithInvalidTokens[deviceIndices[idx]] = struct{}{}
				rErr = ErrInvalidDeviceToken
			} else {
				rErr = errors.Wrapf(resp.Error, "[%v]fcm send failed for %#v", b.sink.applicationYAMLKey, b.notifications[deviceIndices[idx]].notification)
				log.Error(rErr)
			}
			b.devicesFailed[deviceIndices[idx]] = rErr
		} else {
			delete(b.devicesFailed, deviceIndices[idx])
			b.devicesNotified[deviceIndices[idx]] = struct{}{}
		}
	}
	if len(b.notifications) == (len(b.devicesWithInvalidTokens) + len(b.devicesNotified)) {
		return nil
	}

	return errors.Errorf("[%v] at least one token has failed, retrying", b.sink.applicationYAMLKey)
}

func buildFCMMessages(deviceTokenNotifications []*notification, excludeByDeviceIDsSources ...map[int]struct{}) ([]*fcm.Message, []int) {
	fcmMessages := make([]*fcm.Message, 0, len(deviceTokenNotifications))
	indices := make([]int, 0, len(deviceTokenNotifications))
outer:
	for idx, dn := range deviceTokenNotifications {
		if len(excludeByDeviceIDsSources) != 0 {
			for _, excludeByDeviceIDs := range excludeByDeviceIDsSources {
				if _, found := excludeByDeviceIDs[idx]; found {
					continue outer
				}
			}
		}
		indices = append(indices, idx)
		fcmMessages = append(fcmMessages, &fcm.Message{
			Data:  dn.notification.Data,
			Token: string(dn.notification.Target),
			Notification: &fcm.Notification{
				Title:    dn.notification.Title,
				Body:     dn.notification.Body,
				ImageURL: dn.notification.ImageURL,
			},
		})
	}

	return fcmMessages, indices
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
