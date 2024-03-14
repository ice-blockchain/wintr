// SPDX-License-Identifier: ice License 1.0

package http

import "net/http"

func (r *Request) toHttp() *http.Request {
	req := &http.Request{
		Method:           r.Method,
		URL:              r.URL,
		Proto:            r.Proto,
		ProtoMajor:       r.ProtoMajor,
		ProtoMinor:       r.ProtoMinor,
		Header:           r.Header.toHttp(),
		Body:             r.Body,
		GetBody:          r.GetBody,
		ContentLength:    r.ContentLength,
		TransferEncoding: r.TransferEncoding,
		Close:            r.Close,
		Host:             r.Host,
		Form:             r.Form,
		PostForm:         r.PostForm,
		MultipartForm:    r.MultipartForm,
		Trailer:          r.Trailer.toHttp(),
		RemoteAddr:       r.RemoteAddr,
		RequestURI:       r.RequestURI,
		TLS:              r.TLS,
		Cancel:           r.Cancel,
	}
	if r.Response != nil {
		req.Response = r.Response.toHttp(req)
	}
	return req
}

func (r *Response) toHttp(req *http.Request) *http.Response {
	res := &http.Response{
		Status:           r.Status,
		StatusCode:       r.StatusCode,
		Proto:            r.Proto,
		ProtoMajor:       r.ProtoMajor,
		ProtoMinor:       r.ProtoMinor,
		Body:             r.Body,
		ContentLength:    r.ContentLength,
		TransferEncoding: r.TransferEncoding,
		Close:            r.Close,
		Uncompressed:     r.Uncompressed,
		Request:          req,
		TLS:              r.TLS,
	}
	if r.Header != nil {
		req.Header = r.Header.toHttp()
	}
	if r.Trailer != nil {
		req.Trailer = r.Trailer.toHttp()
	}
	return res
}

func (h Header) toHttp() http.Header {
	if len(h) == 0 {
		return make(http.Header)
	}
	return http.Header(h)
}
