// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

func AssertEmailConfirmationCode(ctx context.Context, tb testing.TB, toEmail, expectedCode string, parseMsg func(msg string) string) {
	tb.Helper()
	verificationCodes := []string{}
	// Multiple codes can be sent to the same address, so if 2nd email is not delivered yet - try to fetch massages again.
	foundValidationCode := false
	for !foundValidationCode && ctx.Err() == nil {
		verificationCodes = GetEmailConfirmationCodes(ctx, tb, toEmail, parseMsg)
		for _, code := range verificationCodes {
			if code == expectedCode {
				foundValidationCode = true

				break
			}
		}
		if foundValidationCode {
			break
		}
		time.Sleep(100 * time.Millisecond) //nolint:mnd,gomnd // Not a magic.
	}
	if !foundValidationCode {
		if len(verificationCodes) == 0 {
			require.Fail(tb, "Expected email message, but it was not delivered")
		}
		require.Failf(tb, "Expected %v validation code, but received %v", expectedCode, verificationCodes)
	}
	assert.Contains(tb, verificationCodes, expectedCode)
}

func GetEmailConfirmationCodes(ctx context.Context, tb testing.TB, toEmail string, parseMsg func(msg string) string) []string {
	tb.Helper()
	emailSplitted := strings.Split(toEmail, "@")
	assert.Len(tb, emailSplitted, 2) //nolint:mnd,gomnd // Not a magic.
	login, server := emailSplitted[0], emailSplitted[1]
	mailboxContents := getMailbox(ctx, tb, login, server)
	verificationCodes := make([]string, 0, len(mailboxContents))
	for _, msg := range mailboxContents {
		msgBody := getMessageBody(ctx, tb, login, server, msg)
		verificationCodes = append(verificationCodes, parseMsg(msgBody))
	}

	return verificationCodes
}

func TestingEmail(ctx context.Context, tb testing.TB) string {
	tb.Helper()
	body, status := do1SecMailRequest(ctx, tb, "https://www.1secmail.com/api/v1/?action=genRandomMailbox")
	assert.Equal(tb, 200, status) //nolint:mnd,gomnd // Not a magic.
	var emails []string
	require.NoError(tb, json.UnmarshalContext(ctx, body, &emails))
	assert.Len(tb, emails, 1)

	return emails[0]
}

func getMailbox(ctx context.Context, tb testing.TB, login, server string) []uint64 {
	tb.Helper()
	var messageListData []*struct {
		ID uint64 `json:"id"`
	}
	url := fmt.Sprintf("https://www.1secmail.com/api/v1/?action=getMessages&login=%v&domain=%v", login, server)
	for ctx.Err() == nil && len(messageListData) == 0 {
		messageListBody, messageListStatus := do1SecMailRequest(ctx, tb, url)
		assert.Equal(tb, 200, messageListStatus) //nolint:mnd,gomnd // Not a magic.
		require.NoError(tb, json.UnmarshalContext(ctx, messageListBody, &messageListData))
		time.Sleep(100 * time.Millisecond) //nolint:mnd,gomnd // Not a magic.
	}
	result := make([]uint64, 0, len(messageListData))
	for _, mail := range messageListData {
		result = append(result, mail.ID)
	}

	return result
}

func getMessageBody(ctx context.Context, tb testing.TB, login, server string, msgID uint64) string {
	tb.Helper()
	url := fmt.Sprintf("https://www.1secmail.com/api/v1/?action=readMessage&login=%v&domain=%v&id=%v", login, server, msgID)
	messageInfoBody, messageInfoStatus := do1SecMailRequest(ctx, tb, url)
	assert.Equal(tb, 200, messageInfoStatus) //nolint:mnd,gomnd // Not a magic.
	var messageInfo struct {
		Body string `json:"body"`
	}
	require.NoError(tb, json.UnmarshalContext(ctx, messageInfoBody, &messageInfo))

	return messageInfo.Body
}

func do1SecMailRequest(ctx context.Context, tb testing.TB, url string) (respBody []byte, statusCode int) {
	tb.Helper()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	require.NoError(tb, err)
	//nolint:gosec // .
	client := &http.Client{Transport: &http2.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := client.Do(r)
	require.NoError(tb, err)
	defer func() {
		require.NoError(tb, resp.Body.Close())
	}()

	assert.Equal(tb, 200, resp.StatusCode) //nolint:mnd,gomnd // That's not a magic number.
	b, err := io.ReadAll(resp.Body)
	require.NoError(tb, err)

	return b, resp.StatusCode
}
