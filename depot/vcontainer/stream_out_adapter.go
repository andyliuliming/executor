package vcontainer

import (
	"io"

	"github.com/virtualcloudfoundry/vcontainercommon/vcontainermodels"
)

type StreamOutAdapter struct {
}

func NewStreamOutAdapter(client vcontainermodels.VContainer_StreamOutClient) io.ReadCloser {
	return &StreamOutAdapter{}
}

func (s *StreamOutAdapter) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (s *StreamOutAdapter) Write(p []byte) (n int, err error) {
	return 0, nil
}

func (s *StreamOutAdapter) Close() error {
	return nil
}
