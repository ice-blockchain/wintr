// SPDX-License-Identifier: ice License 1.0

package fixture

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (tc *httpTestClient) Get(
	ctx context.Context,
	tb testing.TB,
	url string,
	headers ...http.Header,
) (respBody string, statusCode int, header http.Header) {
	tb.Helper()

	return tc.doRequest(ctx, tb, http.MethodGet, url, nil, headers...)
}

func (tc *httpTestClient) Delete(
	ctx context.Context,
	tb testing.TB,
	url string,
	headers ...http.Header,
) (respBody string, statusCode int, header http.Header) {
	tb.Helper()

	return tc.doRequest(ctx, tb, http.MethodDelete, url, nil, headers...)
}

func (tc *httpTestClient) Post(
	ctx context.Context,
	tb testing.TB,
	url string,
	body io.Reader,
	headers ...http.Header,
) (respBody string, statusCode int, header http.Header) {
	tb.Helper()

	return tc.doRequest(ctx, tb, http.MethodPost, url, body, headers...)
}

func (tc *httpTestClient) Put(
	ctx context.Context,
	tb testing.TB,
	url string,
	body io.Reader,
	headers ...http.Header,
) (respBody string, statusCode int, header http.Header) {
	tb.Helper()

	return tc.doRequest(ctx, tb, http.MethodPut, url, body, headers...)
}

func (tc *httpTestClient) Patch(
	ctx context.Context,
	tb testing.TB,
	url string,
	body io.Reader,
	headers ...http.Header,
) (respBody string, statusCode int, header http.Header) {
	tb.Helper()

	return tc.doRequest(ctx, tb, http.MethodPatch, url, body, headers...)
}

//nolint:revive // Looks alot better.
func (tc *httpTestClient) doRequest(
	ctx context.Context,
	tb testing.TB,
	method,
	url string,
	body io.Reader,
	headers ...http.Header,
) (respBody string, statusCode int, header http.Header) {
	tb.Helper()

	r, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("https://%v%v", tc.serverAddr, url), body)
	require.NoError(tb, err)
	r.Header = make(http.Header)
	addHeaders(headers, r)
	require.NoError(tb, err)
	resp, err := tc.client.Do(r)
	defer func() { assert.NoError(tb, resp.Body.Close()) }()
	require.NoError(tb, err)
	assert.Equal(tb, "HTTP/2.0", resp.Proto)
	//nolint:gomnd // It's not a magic number, it's the http major version.
	assert.Equal(tb, 2, resp.ProtoMajor)
	assert.Equal(tb, 0, resp.ProtoMinor)

	b, err := io.ReadAll(resp.Body)
	require.NoError(tb, err)
	respBody = string(b)

	return respBody, resp.StatusCode, resp.Header
}

func (tc *httpTestClient) TestSwagger(ctx context.Context, tb testing.TB) {
	tb.Helper()

	if tc.swaggerRoot == "" {
		return
	}
	tc.testSwaggerRoot(ctx, tb)
	tc.testSwaggerIndex(ctx, tb)
	tc.testSwaggerJSON(ctx, tb)
}

func (tc *httpTestClient) testSwaggerRoot(ctx context.Context, tb testing.TB) {
	tb.Helper()

	body, status, headers := tc.Get(ctx, tb, tc.swaggerRoot)
	assert.NotEmpty(tb, len(body))
	assert.Equal(tb, http.StatusOK, status)
	l, err := strconv.Atoi(headers.Get("Content-Length"))
	require.NoError(tb, err)
	require.Greater(tb, l, 0)
	headers.Del("Date")
	headers.Del("Content-Length")
	require.Equal(tb, http.Header{"Content-Type": []string{"text/html; charset=utf-8"}}, headers)
}

func (tc *httpTestClient) testSwaggerIndex(ctx context.Context, tb testing.TB) {
	tb.Helper()

	body, status, headers := tc.Get(ctx, tb, fmt.Sprintf("%v/swagger/index.html", tc.swaggerRoot))
	assert.NotEmpty(tb, len(body))
	assert.Equal(tb, http.StatusOK, status)
	l, err := strconv.Atoi(headers.Get("Content-Length"))
	require.True(tb, err == nil && l > 0)
	headers.Del("Date")
	headers.Del("Content-Length")
	require.Equal(tb, http.Header{"Content-Type": []string{"text/html; charset=utf-8"}}, headers)
}

func (tc *httpTestClient) testSwaggerJSON(ctx context.Context, tb testing.TB) {
	tb.Helper()

	body, status, headers := tc.Get(ctx, tb, fmt.Sprintf("%v/swagger/doc.json", tc.swaggerRoot))
	expectedBuffer := new(bytes.Buffer)
	actualBuffer := new(bytes.Buffer)
	require.NoError(tb, json.Compact(expectedBuffer, []byte(tc.expectedSwaggerJSON)))
	require.NoError(tb, json.Compact(actualBuffer, []byte(body)))
	assert.Equal(tb, expectedBuffer.String(), actualBuffer.String())
	assert.Equal(tb, http.StatusOK, status)
	headers.Del("Date")
	require.Equal(tb, http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, headers)
}

func (tc *httpTestClient) TestHealthCheck(ctx context.Context, tb testing.TB) {
	tb.Helper()

	body, status, headers := tc.Get(ctx, tb, "/health-check", http.Header{"CF-Connecting-IP": []string{"1.2.3.4"}})
	assert.Equal(tb, `{"clientIp":"1.2.3.4"}`, body)
	assert.Equal(tb, http.StatusOK, status)
	headers.Del("Date")
	require.Equal(tb, http.Header{"Content-Length": []string{"22"}, "Content-Type": []string{"application/json; charset=utf-8"}}, headers)
}

func addHeaders(headers []http.Header, r *http.Request) {
	//nolint:revive // False negative.
	if len(headers) != 0 && headers[0] != nil {
		for k, vs := range headers[0] {
			for _, v := range vs {
				r.Header.Add(k, v)
			}
		}
	}
}

func (*httpTestClient) AssertUnauthorized(tb testing.TB, expectedBody, body string, status int, headers http.Header) {
	tb.Helper()

	assert.Equal(tb, expectedBody, body)
	assert.Equal(tb, http.StatusUnauthorized, status)
	l, err := strconv.Atoi(headers.Get("Content-Length"))
	require.NoError(tb, err)
	assert.Greater(tb, l, 0)
	headers.Del("Date")
	headers.Del("Content-Length")
	assert.Equal(tb, http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, headers)
}

func (*httpTestClient) WrapJSONBody(jsonData string) (reqBody io.Reader, contentType string) {
	if jsonData == "" {
		return nil, jsonContentType
	}

	return strings.NewReader(jsonData), jsonContentType
}

func (*httpTestClient) WrapMultipartBody(tb testing.TB, values map[string]any) (reqBody io.Reader, contentType string) {
	tb.Helper()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	defer func() {
		require.NoError(tb, writer.Close())
	}()
	for fieldName, fieldValue := range values {
		switch val := fieldValue.(type) {
		case io.Reader:
			fileName := fieldName
			if file, isFile := val.(fs.File); isFile {
				fileStat, err := file.Stat()
				require.NoError(tb, err)
				fileName = fileStat.Name()
			}
			formFile, err := writer.CreateFormFile(fieldName, fileName)
			require.NoError(tb, err)
			_, err = io.Copy(formFile, val)
			require.NoError(tb, err)
		default:
			require.NoError(tb, writer.WriteField(fieldName, fmt.Sprintf("%v", val)))
		}
	}

	return body, writer.FormDataContentType()
}
