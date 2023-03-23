// SPDX-License-Identifier: ice License 1.0

package server

import (
	"net/http"

	"github.com/pkg/errors"
)

func BadRequest(err error, code string, dataArg ...map[string]any) *Response[ErrorResponse] {
	var data map[string]any
	if len(dataArg) == 1 {
		data = dataArg[0]
	}

	return &Response[ErrorResponse]{
		Data: &ErrorResponse{
			error: err,
			Error: err.Error(),
			Code:  code,
			Data:  data,
		},
		Code: http.StatusBadRequest,
	}
}

func UnprocessableEntity(err error, code string, dataArg ...map[string]any) *Response[ErrorResponse] {
	var data map[string]any
	if len(dataArg) == 1 {
		data = dataArg[0]
	}

	return &Response[ErrorResponse]{
		Data: &ErrorResponse{
			error: err,
			Error: err.Error(),
			Code:  code,
			Data:  data,
		},
		Code: http.StatusUnprocessableEntity,
	}
}

func Conflict(err error, code string, dataArg ...map[string]any) *Response[ErrorResponse] {
	var data map[string]any
	if len(dataArg) == 1 {
		data = dataArg[0]
	}

	return &Response[ErrorResponse]{
		Data: &ErrorResponse{
			error: err,
			Error: err.Error(),
			Code:  code,
			Data:  data,
		},
		Code: http.StatusConflict,
	}
}

func NotFound(err error, code string, dataArg ...map[string]any) *Response[ErrorResponse] {
	var data map[string]any
	if len(dataArg) == 1 {
		data = dataArg[0]
	}

	return &Response[ErrorResponse]{
		Data: &ErrorResponse{
			error: err,
			Error: err.Error(),
			Code:  code,
			Data:  data,
		},
		Code: http.StatusNotFound,
	}
}

func Unexpected(err error) *Response[ErrorResponse] {
	return &Response[ErrorResponse]{
		Code: -1,
		Data: &ErrorResponse{
			error: err,
			Error: err.Error(),
		},
	}
}

func Unauthorized(err error, dataArg ...map[string]any) *Response[ErrorResponse] {
	var data map[string]any
	if len(dataArg) == 1 {
		data = dataArg[0]
	}

	return &Response[ErrorResponse]{
		Code: http.StatusUnauthorized,
		Data: &ErrorResponse{
			error: errors.Wrapf(err, "authorization failed"),
			Error: err.Error(),
			Code:  "INVALID_TOKEN",
			Data:  data,
		},
	}
}

func Forbidden(err error, dataArg ...map[string]any) *Response[ErrorResponse] {
	var data map[string]any
	if len(dataArg) == 1 {
		data = dataArg[0]
	}

	return &Response[ErrorResponse]{
		Code: http.StatusForbidden,
		Data: &ErrorResponse{
			error: err,
			Error: err.Error(),
			Code:  "OPERATION_NOT_ALLOWED",
			Data:  data,
		},
	}
}

func ForbiddenWithCode(err error, code string, dataArg ...map[string]any) *Response[ErrorResponse] {
	var data map[string]any
	if len(dataArg) == 1 {
		data = dataArg[0]
	}

	return &Response[ErrorResponse]{
		Code: http.StatusForbidden,
		Data: &ErrorResponse{
			error: err,
			Error: err.Error(),
			Code:  code,
			Data:  data,
		},
	}
}

func NoContent() *Response[any] {
	return &Response[any]{Code: http.StatusNoContent}
}

func Created[RESP any](resp *RESP) *Response[RESP] {
	return &Response[RESP]{Code: http.StatusCreated, Data: resp}
}

func OK[RESP any](responses ...*RESP) *Response[RESP] {
	var resp *RESP
	if len(responses) == 1 {
		resp = responses[0]
	}

	return &Response[RESP]{Code: http.StatusOK, Data: resp}
}

func (e *ErrorResponse) Fail(err error) *ErrorResponse {
	e.error = err

	return e
}

func (e *ErrorResponse) InternalErr() error {
	return e.error
}
