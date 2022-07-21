// SPDX-License-Identifier: BUSL-1.1

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/goccy/go-reflect"
	"github.com/hashicorp/go-multierror"
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

//nolint:funlen,gofumpt // No idea why.
func RootHandler[REQ any, RESP any](handleRequest func(context.Context, *Request[REQ, RESP]) (*Response[RESP], *Response[ErrorResponse])) func(*gin.Context) {
	return func(ginCtx *gin.Context) {
		ctx, cancel := context.WithTimeout(ginCtx.Request.Context(), cfg.DefaultEndpointTimeout)
		defer cancel()
		if ginCtx.Request.Proto != "HTTP/2.0" {
			log.Warn(fmt.Sprintf("suboptimal http version used for %[1]T", new(REQ)), "expected", "HTTP/2.0", "actual", ginCtx.Request.Proto)
		}
		req := new(Request[REQ, RESP]).init(ginCtx)
		if resp := req.processRequest(); resp != nil {
			log.Error(errors.Wrap(resp.Data.InternalErr(), "endpoint processing failed"), fmt.Sprintf("%[1]T", req.Data), req, "Response", resp)
			ginCtx.JSON(resp.Code, resp.Data)

			return
		}
		if resp := req.authorize(); resp != nil {
			log.Error(errors.Wrap(resp.Data.InternalErr(), "endpoint authentication failed"), fmt.Sprintf("%[1]T", req.Data), req, "Response", resp)
			ginCtx.JSON(resp.Code, resp.Data)

			return
		}
		//nolint:nolintlint // Its gonna come back.
		success, failure := handleRequest(context.WithValue(ctx, requestingUserIDCtxValueKey, req.AuthenticatedUser.ID), req) //nolint:revive,staticcheck // .
		if failure != nil {
			log.Error(errors.Wrap(failure.Data.InternalErr(), "endpoint failed"), fmt.Sprintf("%[1]T", req.Data), req, "Response", failure)
			ginCtx.JSON(req.processErrorResponse(ctx, failure))

			return
		}
		if success.Data != nil {
			ginCtx.JSON(success.Code, success.Data)
		} else {
			ginCtx.Status(success.Code)
		}
	}
}

func (req *Request[REQ, RESP]) init(ginCtx *gin.Context) *Request[REQ, RESP] {
	req.Data = new(REQ)
	req.ClientIP = net.ParseIP(ginCtx.ClientIP())
	req.ginCtx = ginCtx

	return req
}

//nolint:funlen,gocognit,revive // Alot of usecases.
func (req *Request[REQ, RESP]) processTags() {
	elem := reflect.TypeOf(req.Data).Elem()
	if elem.Kind() != reflect.Struct {
		log.Panic("request data's have to be structs")
	}
	const enabled = "true"
	fieldCount := elem.NumField()
	req.requiredFields = make([]string, 0, fieldCount)
	req.bindings = make(map[requestBinding]struct{}, 5) //nolint:gomnd // They're 5 possible values.
	for i := 0; i < fieldCount; i++ {
		field := elem.Field(i)
		tag := field.Tag
		if tag.Get("required") == enabled {
			req.requiredFields = append(req.requiredFields, field.Name)
		}
		if tag.Get("allowUnauthorized") == enabled {
			req.allowUnauthorized = true
		}
		if jsonTag := tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			req.bindings[json] = struct{}{}
		}
		if tag.Get("uri") != "" {
			req.bindings[uri] = struct{}{}
		}
		if tag.Get("header") != "" {
			req.bindings[header] = struct{}{}
		}
		if tag.Get("form") != "" {
			if tag.Get("formMultipart") == "" {
				req.bindings[query] = struct{}{}
			}
		}
		if tag.Get("formMultipart") != "" {
			req.bindings[formMultipart] = struct{}{}
		}
	}
}

func (req *Request[REQ, RESP]) processRequest() *Response[ErrorResponse] {
	req.processTags()
	var errs []error
	for b := range req.bindings {
		switch b {
		case json:
			errs = append(errs, req.ginCtx.ShouldBindJSON(req.Data))
		case uri:
			errs = append(errs, req.ginCtx.ShouldBindUri(req.Data))
		case query:
			errs = append(errs, req.ginCtx.ShouldBindQuery(req.Data))
		case header:
			errs = append(errs, req.ginCtx.ShouldBindHeader(req.Data))
		case formMultipart:
			errs = append(errs, req.ginCtx.ShouldBindWith(req.Data, binding.FormMultipart))
		}
	}
	if err := multierror.Append(nil, errs...).ErrorOrNil(); err != nil {
		return UnprocessableEntity(errors.Wrapf(err, "binding failed"), "STRUCTURE_VALIDATION_FAILED")
	}

	return req.validate()
}

func (req *Request[REQ, RESP]) validate() *Response[ErrorResponse] {
	if len(req.requiredFields) == 0 {
		return nil
	}
	value := reflect.ValueOf(req.Data).Elem()
	requiredFields := make([]string, 0, len(req.requiredFields))
	for _, field := range req.requiredFields {
		if value.FieldByName(field).IsZero() {
			requiredFields = append(requiredFields, field)
		}
	}
	if len(requiredFields) == 0 {
		return nil
	}

	return UnprocessableEntity(errors.Errorf("properties `%v` are required", strings.Join(requiredFields, ",")), "MISSING_PROPERTIES")
}

func (req *Request[REQ, RESP]) authorize() (errResp *Response[ErrorResponse]) {
	if req.allowUnauthorized {
		defer func() {
			errResp = nil
		}()
	}

	tk, err := token.NewToken(strings.TrimPrefix(req.ginCtx.GetHeader("Authorization"), "Bearer "))
	if err != nil {
		return Unauthorized(err)
	}
	if err = tk.Validate(); err != nil {
		return Unauthorized(err)
	}
	req.AuthenticatedUser.ID = tk.GetIssuer()

	userID := strings.Trim(req.ginCtx.Param("userId"), " ")
	if userID != "" &&
		userID != "-" &&
		req.AuthenticatedUser.ID != userID &&
		(req.ginCtx.Request.Method != http.MethodGet || !strings.HasSuffix(req.ginCtx.Request.URL.Path, userID)) {
		return Forbidden(errors.Errorf("operation not allowed. uri>%v!=token>%v", userID, req.AuthenticatedUser.ID))
	}

	return nil
}

func (req *Request[REQ, RESP]) processErrorResponse(ctx context.Context, failure *Response[ErrorResponse]) (int, *ErrorResponse) {
	err := failure.Data.InternalErr()
	if errors.Is(err, req.ginCtx.Request.Context().Err()) {
		return http.StatusServiceUnavailable, &ErrorResponse{Error: "service is shutting down"}
	}
	if errors.Is(err, ctx.Err()) {
		return http.StatusGatewayTimeout, &ErrorResponse{Error: "request timed out"}
	}
	if failure.Code <= 0 {
		return http.StatusInternalServerError, &ErrorResponse{Error: "oops, something went wrong"}
	}

	return failure.Code, failure.Data
}
