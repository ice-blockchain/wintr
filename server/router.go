// SPDX-License-Identifier: BUSL-1.1

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/magiclabs/magic-admin-go/token"
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/log"
)

//nolint:gochecknoinits // Because we want to set it up globally.
func init() {
	if err := os.Setenv("TZ", ""); err != nil {
		log.Panic(err)
	}
}

func ScanRequest(req ParsedRequest, bindings ...func(obj interface{}) error) *Response {
	for _, f := range bindings {
		if err := f(req); err != nil {
			return &Response{
				Code: http.StatusUnprocessableEntity,
				Data: ErrorResponse{
					error: errors.Wrapf(err, "binding failed"),
					Error: err.Error(),
					Code:  "STRUCTURE_VALIDATION_FAILED",
				},
			}
		}
	}

	return req.Validate()
}

func BadRequest(err error, code string, data ...map[string]interface{}) *Response {
	var d map[string]interface{}
	if len(data) == 1 {
		d = data[0]
	}

	return &Response{
		Data: ErrorResponse{
			error: err,
			Error: err.Error(),
			Code:  code,
			Data:  d,
		},
		Code: http.StatusBadRequest,
	}
}

func Conflict(err error, code string, data ...map[string]interface{}) *Response {
	var d map[string]interface{}
	if len(data) == 1 {
		d = data[0]
	}

	return &Response{
		Data: ErrorResponse{
			error: err,
			Error: err.Error(),
			Code:  code,
			Data:  d,
		},
		Code: http.StatusConflict,
	}
}

func NotFound(err error, code string, data ...map[string]interface{}) *Response {
	var d map[string]interface{}
	if len(data) == 1 {
		d = data[0]
	}

	return &Response{
		Data: ErrorResponse{
			error: err,
			Error: err.Error(),
			Code:  code,
			Data:  d,
		},
		Code: http.StatusNotFound,
	}
}

func Unexpected(err error) Response {
	return Response{Data: err}
}

func Unauthorized(err error, data ...map[string]interface{}) *Response {
	var d map[string]interface{}
	if len(data) == 1 {
		d = data[0]
	}

	return &Response{
		Code: http.StatusUnauthorized,
		Data: ErrorResponse{
			error: errors.Wrapf(err, "authorization failed"),
			Error: err.Error(),
			Code:  "INVALID_TOKEN",
			Data:  d,
		},
	}
}

func Forbidden(err error, data ...map[string]interface{}) *Response {
	var d map[string]interface{}
	if len(data) == 1 {
		d = data[0]
	}

	return &Response{
		Code: http.StatusForbidden,
		Data: ErrorResponse{
			error: err,
			Error: err.Error(),
			Code:  "OPERATION_NOT_ALLOWED",
			Data:  d,
		},
	}
}

func NoContent() Response {
	return Response{Code: http.StatusNoContent}
}

func Created(data interface{}) Response {
	return Response{Code: http.StatusCreated, Data: data}
}

func OK(b ...interface{}) Response {
	var data interface{}
	if len(b) == 1 {
		data = b[0]
	}

	return Response{Code: http.StatusOK, Data: data}
}

func ShouldBindClientIP(c *gin.Context) func(obj interface{}) error {
	return func(obj interface{}) error {
		if ip, isOk := obj.(ClientIPSetGetter); isOk {
			ip.SetClientIP(net.ParseIP(c.ClientIP()))
		}

		return nil
	}
}

func ShouldBindAuthenticatedUser(c *gin.Context) func(obj interface{}) error {
	return func(obj interface{}) error {
		if authenticatedUserSetGetter, isOk := obj.(AuthenticatedUserSetGetter); isOk {
			if authenticatedUser, exists := c.Get(authenticatedUserGinCtxKey); exists {
				authenticatedUserSetGetter.SetAuthenticatedUser(authenticatedUser.(AuthenticatedUser))
			} else if condition, ok := obj.(AuthenticatedUserCondition); !ok || condition.ShouldAuthenticateUser(c) {
				return ErrNotAuthenticated
			}
		}

		return nil
	}
}

func RequiredStrings(fields map[string]string) *Response {
	var failedFields []string
	for fieldName, fieldValue := range fields {
		if strings.TrimSpace(fieldValue) == "" {
			failedFields = append(failedFields, fmt.Sprintf("`%v`", fieldName))
		}
	}

	if len(failedFields) == 0 {
		return nil
	}
	sort.Slice(failedFields, func(i, j int) bool { return failedFields[i] < failedFields[j] })

	err := errors.Errorf("properties %v are required", strings.Join(failedFields, ","))

	return BadRequest(err, "MISSING_PROPERTIES")
}

func RootHandler[T ParsedRequest](r func() T, handleRequest func(context.Context, T) Response) func(*gin.Context) {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.DefaultEndpointTimeout)
		defer cancel()
		req := r()
		if c.Request.Proto != "HTTP/2.0" {
			log.Warn(fmt.Sprintf("suboptimal http version used for %[1]T", req), "expected", "HTTP/2.0", "actual", c.Request.Proto)
		}

		if resp := authorize(req, c); resp != nil {
			err := errors.Wrap(resp.Data.(ErrorResponse).error, "endpoint authentication failed")
			log.Error(err, fmt.Sprintf("%[1]T", req), req, "Response", resp)
			c.JSON(resp.Code, resp.Data)

			return
		}

		if resp := ScanRequest(req, req.Bindings(c)...); resp != nil {
			err := errors.Wrap(resp.Data.(ErrorResponse).error, "endpoint binding failed")
			log.Error(err, fmt.Sprintf("%[1]T", req), req, "Response", resp)
			c.JSON(resp.Code, resp.Data)

			return
		}

		if resp := handleRequest(ctx, req); resp.Data != nil {
			c.JSON(handleData(ctx, c.Request.Context(), req, resp))
		} else {
			c.Status(resp.Code)
		}
	}
}

func authorize(req ParsedRequest, c *gin.Context) *Response {
	if _, ok := req.(AuthenticatedUserSetGetter); !ok {
		return nil
	}
	if condition, ok := req.(AuthenticatedUserCondition); ok && !condition.ShouldAuthenticateUser(c) {
		return nil
	}
	tk, err := token.NewToken(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
	if err != nil {
		return Unauthorized(err)
	}
	err = tk.Validate()
	if err != nil {
		return Unauthorized(err)
	}

	c.Set(authenticatedUserGinCtxKey, AuthenticatedUser{
		ID: tk.GetIssuer(),
	})

	return nil
}

func handleData(ctx, baseCtx context.Context, req ParsedRequest, resp Response) (int, interface{}) {
	if e, isOk := resp.Data.(error); isOk {
		log.Error(errors.Wrap(e, "endpoint failed"), fmt.Sprintf("%[1]T", req), req, "Response", resp)
		if _, isOk = resp.Data.(ErrorResponse); !isOk {
			return handleUnexpectedError(ctx, baseCtx, resp.Code, e)
		}
	}

	return resp.Code, resp.Data
}

func handleUnexpectedError(ctx, baseCtx context.Context, code int, e error) (int, interface{}) {
	if errors.Is(e, baseCtx.Err()) {
		return http.StatusServiceUnavailable, ErrorResponse{Error: "service is shutting down"}
	}
	if errors.Is(e, ctx.Err()) {
		return http.StatusGatewayTimeout, ErrorResponse{Error: "request timed out"}
	}
	if code == 0 {
		return http.StatusInternalServerError, ErrorResponse{Error: ErrSomethingWentWrong.Error()}
	}

	return code, ErrorResponse{Error: e.Error()}
}

func (e ErrorResponse) Fail(err error) ErrorResponse {
	e.error = err

	return e
}

func (e ErrorResponse) InternalErr() error {
	return e.error
}
