package vcontainer

import (
	"io"

	"code.cloudfoundry.org/lager"

	"github.com/virtualcloudfoundry/vcontainercommon/verrors"

	"github.com/virtualcloudfoundry/vcontainercommon/vcontainermodels"
)

type StreamOutAdapter struct {
	logger lager.Logger
	client vcontainermodels.VContainer_StreamOutClient
}

func NewStreamOutAdapter(logger lager.Logger, client vcontainermodels.VContainer_StreamOutClient) io.ReadCloser {
	return &StreamOutAdapter{
		logger: logger,
		client: client,
	}
}

func (s *StreamOutAdapter) Read(p []byte) (n int, err error) {
	response, err := s.client.Recv()
	if err != nil {
		return 0, verrors.New("stream-out-adapter-read-failed")
	}
	copy(p, response.Content)
	return len(response.Content), nil
}

func (s *StreamOutAdapter) Write(p []byte) (n int, err error) {
	return 0, verrors.New("stream-out-adapter-writer-not-implemented")
}

func (s *StreamOutAdapter) Close() error {
	err := s.client.CloseSend()
	if err != nil {
		s.logger.Error("stream-out-adapter-client-close-failed", err)
		return verrors.New("stream-out-adapter-close-failed")
	}
	return nil
}
