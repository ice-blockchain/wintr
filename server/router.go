// SPDX-License-Identifier: ice License 1.0

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
	"github.com/pkg/errors"

	"github.com/ice-blockchain/wintr/auth"
	"github.com/ice-blockchain/wintr/log"
)

//nolint:gochecknoinits // Because we want to set it up globally.
func init() {
	if err := os.Setenv("TZ", ""); err != nil {
		log.Panic(err)
	}
}

//nolint:funlen // .
func RootHandler[REQ, RESP any](handleRequest func(context.Context, *Request[REQ, RESP]) (*Response[RESP], *Response[ErrorResponse])) func(*gin.Context) {
	return func(ginCtx *gin.Context) {
		ctx, cancel := context.WithTimeout(ginCtx.Request.Context(), cfg.DefaultEndpointTimeout)
		defer cancel()
		if ginCtx.Request.Proto != "HTTP/2.0" {
			log.Warn(fmt.Sprintf("suboptimal http version used for %[1]T", new(REQ)), "expected", "HTTP/2.0", "actual", ginCtx.Request.Proto)
		}
		req := new(Request[REQ, RESP]).init(ginCtx)
		if err := req.processRequest(); err != nil {
			log.Error(errors.Wrap(err.Data.InternalErr(), "endpoint processing failed"), fmt.Sprintf("%[1]T", req.Data), req, "Response", err)
			ginCtx.JSON(err.Code, err.Data)

			return
		}
		if err := req.authorize(ctx); err != nil {
			log.Error(errors.Wrap(err.Data.InternalErr(), "endpoint authentication failed"), fmt.Sprintf("%[1]T", req.Data), req, "Response", err)
			ginCtx.JSON(err.Code, err.Data)

			return
		}
		reqCtx := context.WithValue(ctx, requestingUserIDCtxValueKey, req.AuthenticatedUser.UserID) //nolint:staticcheck,revive // .
		success, failure := handleRequest(reqCtx, req)
		if failure != nil {
			log.Error(errors.Wrap(failure.Data.InternalErr(), "endpoint failed"), fmt.Sprintf("%[1]T", req.Data), req, "Response", failure)
			ginCtx.JSON(req.processErrorResponse(ctx, failure))

			return
		}
		for k, v := range success.Headers {
			ginCtx.Header(k, v)
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
	req.bindings = make(map[requestBinding]struct{}, 5) //nolint:mnd,gomnd // They're 5 possible values.
	for i := range fieldCount {
		field := elem.Field(i)
		tag := field.Tag
		if tag.Get("required") == enabled {
			req.requiredFields = append(req.requiredFields, field.Name)
		}
		if tag.Get("allowUnauthorized") == enabled {
			req.allowUnauthorized = true
		}
		if tag.Get("allowForbiddenGet") == enabled {
			req.allowForbiddenGet = true
		}
		if tag.Get("allowForbiddenWriteOperation") == enabled {
			req.allowForbiddenWriteOperation = true
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
		switch b { //nolint:revive // .
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

//nolint:gocyclo,revive,cyclop,gocognit // .
func (req *Request[REQ, RESP]) authorize(ctx context.Context) (errResp *Response[ErrorResponse]) {
	userID := strings.Trim(req.ginCtx.Param("userId"), " ")
	if req.allowUnauthorized {
		defer func() {
			if ((req.ginCtx.Request.Method == http.MethodGet || req.ginCtx.Request.Method == http.MethodPost) && userID == "") || userID == "-" {
				errResp = nil
			}
		}()
	}
	authToken := strings.TrimPrefix(req.ginCtx.GetHeader("Authorization"), "Bearer ")
	token, err := Auth(ctx).VerifyToken(ctx, authToken)
	if err != nil {
		if errors.Is(err, auth.ErrForbidden) {
			return Forbidden(err)
		}

		return Unauthorized(err)
	}
	metadataHeader := req.ginCtx.GetHeader("X-Account-Metadata")
	if token, err = Auth(ctx).ModifyTokenWithMetadata(token, metadataHeader); err != nil {
		return Unauthorized(err)
	}
	req.AuthenticatedUser.Token = *token
	req.AuthenticatedUser.Language = req.ginCtx.GetHeader(languageHeader)
	if userID != "" && userID != "-" && req.AuthenticatedUser.UserID != userID &&
		((!req.allowForbiddenWriteOperation && req.ginCtx.Request.Method != http.MethodGet) ||
			(req.ginCtx.Request.Method == http.MethodGet && !req.allowForbiddenGet && !strings.HasSuffix(req.ginCtx.Request.URL.Path, userID))) {
		return Forbidden(errors.Errorf("operation not allowed. uri>%v!=token>%v", userID, req.AuthenticatedUser.UserID))
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

func Auth(ctx context.Context) auth.Client {
	return ctx.Value(authClientCtxValueKey).(auth.Client) //nolint:forcetypeassert,revive // We know for sure.
}
