// SPDX-License-Identifier: ice License 1.0

package push

import (
	"fmt"
	stdlibtime "time"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
)

func (s *pushNotificationsCollectingSink) startProcessing() {
	s.wg.Add(1)
	log.Info("pushNotificationsCollectingSink, starting processing", "package", s.applicationYAMLKey)
	defer s.wg.Done()
	defer log.Info("pushNotificationsCollectingSink, stopping processing", "package", s.applicationYAMLKey)

	for notifications := range s.bufferedNotificationsChan {
		s.newNotificationBatch(notifications...).process()
	}
}

func (s *pushNotificationsCollectingSink) stop() {
	s.mx.Lock()
	defer s.mx.Unlock()
	if !s.closed {
		if len(s.bufferedNotifications) > 0 {
			log.Info(fmt.Sprintf("STOPPING with %v bufferedNotifications", len(s.bufferedNotifications)))
			for i := 0; i < len(s.bufferedNotifications)/fcmSendAllBatchSize; i++ {
				s.bufferedNotificationsChan <- s.bufferedNotifications[i*fcmSendAllBatchSize : (i+1)*fcmSendAllBatchSize]
			}
			remainingElements := len(s.bufferedNotifications) % fcmSendAllBatchSize
			s.bufferedNotificationsChan <- s.bufferedNotifications[len(s.bufferedNotifications)-remainingElements:]
			s.bufferedNotifications = make([]*notification, 0, fcmSendAllBatchSize)
			s.resetAt = time.Now()
		}
		close(s.bufferedNotificationsChan)
		s.wg.Wait()
		s.closed = true
	}
}

func (s *pushNotificationsCollectingSink) accept(notif *Notification[DeviceToken], responseChan chan<- error) { //nolint:funlen,gocognit,revive // .
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.closed {
		if notif != nil {
			log.Warn("notification received after closing", "notification", notif, "package", s.applicationYAMLKey)
			s.newNotificationBatch(&notification{notification: notif, responder: responseChan}).process()
		}

		return
	}
	now := time.Now()
	if cap(s.bufferedNotifications) == 0 {
		s.bufferedNotifications = make([]*notification, 0, fcmSendAllBatchSize)
	}
	if s.resetAt == nil {
		s.resetAt = now
	}
	if notif != nil {
		s.bufferedNotifications = append(s.bufferedNotifications, &notification{notification: notif, responder: responseChan})
	}
	if len(s.bufferedNotifications) >= fcmSendAllBatchSize || now.Sub(*s.resetAt.Time) >= fcmSendAllBufferingDeadline {
		if len(s.bufferedNotifications) > 0 {
			for i := 0; i < len(s.bufferedNotifications)/fcmSendAllBatchSize; i++ {
				s.bufferedNotificationsChan <- s.bufferedNotifications[i*fcmSendAllBatchSize : (i+1)*fcmSendAllBatchSize]
			}
			remainingElements := len(s.bufferedNotifications) % fcmSendAllBatchSize
			s.bufferedNotifications = s.bufferedNotifications[len(s.bufferedNotifications)-remainingElements:]
			if now.Sub(*s.resetAt.Time) >= fcmSendAllBufferingDeadline && len(s.bufferedNotifications) > 0 {
				s.bufferedNotificationsChan <- s.bufferedNotifications
				s.bufferedNotifications = make([]*notification, 0, fcmSendAllBatchSize)
			}
		}
		s.resetAt = now
	}
}

func (s *pushNotificationsCollectingSink) monitorSlowProcessing() {
	log.Info("pushNotificationsCollectingSink, starting monitorSlowProcessing", "package", s.applicationYAMLKey)
	ticker := stdlibtime.NewTicker(fcmSendAllSlowProcessingMonitoringTickerDeadline)
	defer log.Info("pushNotificationsCollectingSink, stopped monitorSlowProcessing", "package", s.applicationYAMLKey)
	defer ticker.Stop()

	for !s.closed {
		<-ticker.C
		s.accept(nil, nil)
	}
}
