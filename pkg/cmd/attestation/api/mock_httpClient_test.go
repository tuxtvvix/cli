package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/cli/cli/v2/pkg/cmd/attestation/test/data"
	"github.com/golang/snappy"
	"github.com/stretchr/testify/mock"
)

type mockHttpClient struct {
	mock.Mock
}

func (m *mockHttpClient) Get(url string) (*http.Response, error) {
	m.On("OnGetSuccess").Return()
	m.MethodCalled("OnGetSuccess")

	var compressed []byte
	compressed = snappy.Encode(compressed, data.SigstoreBundleRaw)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(compressed)),
	}, nil
}

type failHttpClient struct {
	mock.Mock
}

func (m *failHttpClient) Get(url string) (*http.Response, error) {
	m.On("OnGetFail").Return()
	m.MethodCalled("OnGetFail")

	return &http.Response{
		StatusCode: 500,
	}, fmt.Errorf("failed to fetch with %s", url)
}

type failAfterNCallsHttpClient struct {
	mock.Mock
	FailOnCallN              int
	FailOnAllSubsequentCalls bool
	NumCalls                 int
}

func (m *failAfterNCallsHttpClient) Get(url string) (*http.Response, error) {
	m.On("OnGetFailAfterNCalls").Return()

	m.NumCalls++

	if m.NumCalls == m.FailOnCallN || (m.NumCalls > m.FailOnCallN && m.FailOnAllSubsequentCalls) {
		m.MethodCalled("OnGetFailAfterNCalls")
		return &http.Response{
			StatusCode: 500,
		}, nil
	}

	m.MethodCalled("OnGetFailAfterNCalls")
	var compressed []byte
	compressed = snappy.Encode(compressed, data.SigstoreBundleRaw)
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewReader(compressed)),
	}, nil
}
