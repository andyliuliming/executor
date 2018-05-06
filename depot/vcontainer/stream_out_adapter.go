package vcontainer

import (
	"io"

	"code.cloudfoundry.org/lager"

	"github.com/virtualcloudfoundry/vcontainercommon/verrors"

	"github.com/virtualcloudfoundry/vcontainercommon/vcontainermodels"
)

type StreamOutAdapter struct {
	logger          lager.Logger
	client          vcontainermodels.VContainer_StreamOutClient
	currentResponse []byte
	currentIndex    int
}

func NewStreamOutAdapter(logger lager.Logger, client vcontainermodels.VContainer_StreamOutClient) io.ReadCloser {
	return &StreamOutAdapter{
		logger: logger,
		client: client,
	}
}

func (s *StreamOutAdapter) Read(p []byte) (n int, err error) {
	if s.currentResponse != nil || s.currentIndex == len(s.currentResponse)-1 {
		response, err := s.client.Recv()

		if err != nil {
			if err == io.EOF {
				s.logger.Info("stream-out-adapter-read-recv-eof-got")
				return 0, io.EOF
			} else {
				s.logger.Error("stream-out-adapter-read-recv-failed", err)
				return 0, verrors.New("stream-out-adapter-read-failed")
			}
		}
		s.logger.Info("stream-out-adapter-read", lager.Data{
			"buffer_size": len(p),
			"content_len": len(response.Content)})
		s.currentResponse = response.Content
	}

	bufferSize := len(p)
	if leftSize := len(s.currentResponse) - s.currentIndex; leftSize < bufferSize {
		copy(p, s.currentResponse[s.currentIndex:])
		return leftSize, nil
	} else {
		copy(p, s.currentResponse[s.currentIndex:])
		return bufferSize, nil
	}
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
