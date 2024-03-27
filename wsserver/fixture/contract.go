package fixture

import (
	_ "embed"
	h2ec "github.com/ice-blockchain/go/src/net/http"
	"github.com/ice-blockchain/wintr/wsserver"
	"github.com/ice-blockchain/wintr/wsserver/internal"
	"io"
	"net"
	"sync"
	"sync/atomic"
	stdlibtime "time"
)

var (
	//go:embed .testdata/localhost.crt
	localhostCrt string
	//go:embed .testdata/localhost.key
	localhostKey string
)

type (
	MockService struct {
		server         wsserver.Server
		closed         bool
		closedMx       sync.Mutex
		processingFunc func(writer wsserver.WSWriter, in string) error
		ReaderExited   atomic.Uint64
	}
	Client interface {
		Received
		wsserver.WSWriter
	}
	Received interface {
		Received() <-chan []byte
	}
)

const (
	applicationYamlKey = "self"
	wtCapsuleStream    = 0x190B4D3B
	wtCapsuleStreamFin = 0x190B4D3C
)

type (
	wsocketClient struct {
		conn          net.Conn
		out           chan wsWrite
		closeChannel  chan struct{}
		closed        bool
		closeMx       sync.Mutex
		writeTimeout  stdlibtime.Duration
		readTimeout   stdlibtime.Duration
		inputMessages chan []byte
	}
	wtransportClient struct {
		wt            *internal.WebtransportAdapter
		inputMessages chan []byte
		closed        bool
		closedMx      sync.Mutex
	}
	http2ClientStream struct {
		w    *io.PipeWriter
		resp *h2ec.Response
	}
	http2WebtransportWrapper struct {
		conn     *http2ClientStream
		streamID uint32
	}
	wsWrite struct {
		data   []byte
		opCode int
	}
)
