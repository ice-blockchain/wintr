// SPDX-License-Identifier: ice License 1.0

package http

import (
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/quicvarint"
)

// RFC 9297: https://datatracker.ietf.org/doc/rfc9297/
// To be used in webtransport over http2

const headerCapsuleProtocol = "Capsule-Protocol"

func (rw *http2responseWriter) WriteCapsule(typ uint32, capsule Capsule) error {
	rw.Header().Add(headerCapsuleProtocol, strconv.FormatBool(true))
	// https://lists.w3.org/Archives/Public/ietf-http-wg/2018AprJun/0258.html
	wErr := http3.WriteCapsule(rw.rws.bw, http3.CapsuleType(typ), capsule.Serialize())
	return multierror.Append(
		wErr,
		rw.FlushError(),
	).ErrorOrNil()

}

type Capsule interface {
	Serialize() []byte
	Deserialize(data quicvarint.Reader) error
}
