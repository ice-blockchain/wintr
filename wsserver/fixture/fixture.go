package fixture

import (
	"context"
	"github.com/ice-blockchain/wintr/wsserver"
)

func NewTestServer(ctx context.Context, cancel context.CancelFunc, processingFunc func(w wsserver.WSWriter, in string) error) *MockService {
	service := newMockService(processingFunc)
	server := wsserver.New(service, applicationYamlKey)
	service.server = server
	go service.server.ListenAndServe(ctx, cancel)

	return service
}
func newMockService(processingFunc func(w wsserver.WSWriter, in string) error) *MockService {
	return &MockService{processingFunc: processingFunc}
}

func (m *MockService) Read(ctx context.Context, w wsserver.WS) {
	defer func() {
		m.ReaderExited.Add(1)
	}()
	for ctx.Err() == nil {
		_, msg, err := w.ReadMessage()
		if err != nil {
			break
		}
		if len(msg) > 0 {
			m.processingFunc(w, string(msg))
		}
	}
}
func (m *MockService) Init(ctx context.Context, cancel context.CancelFunc) {
}

func (m *MockService) Close(ctx context.Context) error {
	return nil
}

func (m *MockService) RegisterRoutes(r *wsserver.Router) {

}
