package api

import (
	"testing"

	"github.com/cli/cli/v2/pkg/cmd/attestation/io"
	"github.com/cli/cli/v2/pkg/cmd/attestation/test/data"

	"github.com/stretchr/testify/require"
)

const (
	testRepo   = "github/example"
	testOwner  = "github"
	testDigest = "sha256:12313213"
)

func NewClientWithMockGHClient(hasNextPage bool) Client {
	fetcher := mockDataGenerator{
		NumAttestations: 5,
	}
	l := io.NewTestHandler()

	httpClient := &mockHttpClient{}

	if hasNextPage {
		return &LiveClient{
			githubAPI: mockAPIClient{
				OnRESTWithNext: fetcher.OnRESTSuccessWithNextPage,
			},
			httpClient: httpClient,
			logger:     l,
		}
	}

	return &LiveClient{
		githubAPI: mockAPIClient{
			OnRESTWithNext: fetcher.OnRESTSuccess,
		},
		httpClient: httpClient,
		logger:     l,
	}
}

func TestGetByDigest(t *testing.T) {
	c := NewClientWithMockGHClient(false)
	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, DefaultLimit)
	require.NoError(t, err)

	require.Equal(t, 5, len(attestations))
	bundle := (attestations)[0].Bundle
	require.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle.v0.3+json")

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, DefaultLimit)
	require.NoError(t, err)

	require.Equal(t, 5, len(attestations))
	bundle = (attestations)[0].Bundle
	require.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle.v0.3+json")
}

func TestGetByDigestGreaterThanLimit(t *testing.T) {
	c := NewClientWithMockGHClient(false)

	limit := 3
	// The method should return five results when the limit is not set
	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, limit)
	require.NoError(t, err)

	require.Equal(t, 3, len(attestations))
	bundle := (attestations)[0].Bundle
	require.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle.v0.3+json")

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, limit)
	require.NoError(t, err)

	require.Equal(t, len(attestations), limit)
	bundle = (attestations)[0].Bundle
	require.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle.v0.3+json")
}

func TestGetByDigestWithNextPage(t *testing.T) {
	c := NewClientWithMockGHClient(true)
	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, DefaultLimit)
	require.NoError(t, err)

	require.Equal(t, len(attestations), 10)
	bundle := (attestations)[0].Bundle
	require.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle.v0.3+json")

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, DefaultLimit)
	require.NoError(t, err)

	require.Equal(t, len(attestations), 10)
	bundle = (attestations)[0].Bundle
	require.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle.v0.3+json")
}

func TestGetByDigestGreaterThanLimitWithNextPage(t *testing.T) {
	c := NewClientWithMockGHClient(true)

	limit := 7
	// The method should return five results when the limit is not set
	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, limit)
	require.NoError(t, err)

	require.Equal(t, len(attestations), limit)
	bundle := (attestations)[0].Bundle
	require.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle.v0.3+json")

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, limit)
	require.NoError(t, err)

	require.Equal(t, len(attestations), limit)
	bundle = (attestations)[0].Bundle
	require.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle.v0.3+json")
}

func TestGetByDigest_NoAttestationsFound(t *testing.T) {
	fetcher := mockDataGenerator{
		NumAttestations: 5,
	}

	httpClient := &mockHttpClient{}
	c := LiveClient{
		githubAPI: mockAPIClient{
			OnRESTWithNext: fetcher.OnRESTWithNextNoAttestations,
		},
		httpClient: httpClient,
		logger:     io.NewTestHandler(),
	}

	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, DefaultLimit)
	require.Error(t, err)
	require.IsType(t, ErrNoAttestationsFound, err)
	require.Nil(t, attestations)

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, DefaultLimit)
	require.Error(t, err)
	require.IsType(t, ErrNoAttestationsFound, err)
	require.Nil(t, attestations)
}

func TestGetByDigest_Error(t *testing.T) {
	fetcher := mockDataGenerator{
		NumAttestations: 5,
	}

	c := LiveClient{
		githubAPI: mockAPIClient{
			OnRESTWithNext: fetcher.OnRESTWithNextError,
		},
		logger: io.NewTestHandler(),
	}

	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, DefaultLimit)
	require.Error(t, err)
	require.Nil(t, attestations)

	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, DefaultLimit)
	require.Error(t, err)
	require.Nil(t, attestations)
}

func TestFetchBundleFromAttestations_BundleURL(t *testing.T) {
	httpClient := &mockHttpClient{}
	client := LiveClient{
		httpClient: httpClient,
		logger:     io.NewTestHandler(),
	}

	att1 := makeTestAttestation()
	att1.Bundle = nil
	att2 := makeTestAttestation()
	att2.Bundle = nil
	// zero out the bundle field so it tries fetching by URL
	attestations := []*Attestation{&att1, &att2}
	fetched, err := client.fetchBundleFromAttestations(attestations)
	require.NoError(t, err)
	require.Len(t, fetched, 2)
	require.NotNil(t, "application/vnd.dev.sigstore.bundle.v0.3+json", fetched[0].Bundle.GetMediaType())
	httpClient.AssertNumberOfCalls(t, "OnGetSuccess", 2)
}

func TestFetchBundleFromAttestations_InvalidAttestation(t *testing.T) {
	httpClient := &mockHttpClient{}
	client := LiveClient{
		httpClient: httpClient,
		logger:     io.NewTestHandler(),
	}

	att1 := Attestation{}
	attestations := []*Attestation{&att1}
	fetched, err := client.fetchBundleFromAttestations(attestations)
	require.Error(t, err)
	require.Nil(t, fetched, 2)
}

func TestFetchBundleFromAttestations_FetchByURLFail(t *testing.T) {
	mockHTTPClient := &failHttpClient{}
	// failAfterOneCallHttpClient

	c := &LiveClient{
		httpClient: mockHTTPClient,
		logger:     io.NewTestHandler(),
	}

	a := makeTestAttestation()
	attestations := []*Attestation{&a}
	bundle, err := c.fetchBundleFromAttestations(attestations)
	require.Error(t, err)
	require.Nil(t, bundle)
}

func TestFetchBundleFromAttestations_FailWithGetErr(t *testing.T) {
	mockHTTPClient := &failHttpClient{}

	c := &LiveClient{
		httpClient: mockHTTPClient,
		logger:     io.NewTestHandler(),
	}

	a := makeTestAttestation()
	a.Bundle = nil
	attestations := []*Attestation{&a}
	bundle, err := c.fetchBundleFromAttestations(attestations)
	require.Error(t, err)
	require.Nil(t, bundle)
	mockHTTPClient.AssertNumberOfCalls(t, "OnGetFail", 1)
}

func TestFetchBundleFromAttestations_FetchByURL5XX_Resp(t *testing.T) {
	mockHTTPClient := &failAfterNCallsHttpClient{
		FailOnCallN:              1,
		FailOnAllSubsequentCalls: true,
	}

	c := &LiveClient{
		httpClient: mockHTTPClient,
		logger:     io.NewTestHandler(),
	}

	a := makeTestAttestation()
	attestations := []*Attestation{&a}
	bundle, err := c.fetchBundleFromAttestations(attestations)
	require.Error(t, err)
	require.Nil(t, bundle)
}

func TestFetchBundleFromAttestations_FallbackToBundleField(t *testing.T) {
	mockHTTPClient := &mockHttpClient{}

	c := &LiveClient{
		httpClient: mockHTTPClient,
		logger:     io.NewTestHandler(),
	}

	a := Attestation{Bundle: data.SigstoreBundle(t)}
	attestations := []*Attestation{&a}
	fetched, err := c.fetchBundleFromAttestations(attestations)
	require.NoError(t, err)
	require.Equal(t, "application/vnd.dev.sigstore.bundle.v0.3+json", fetched[0].Bundle.GetMediaType())
	mockHTTPClient.AssertNotCalled(t, "OnGetSuccess")
}

func TestGetBundle(t *testing.T) {
	mockHTTPClient := &mockHttpClient{}

	c := &LiveClient{
		httpClient: mockHTTPClient,
		logger:     io.NewTestHandler(),
	}

	b, err := c.getBundle("whatever")
	require.NoError(t, err)
	require.Equal(t, "application/vnd.dev.sigstore.bundle.v0.3+json", b.GetMediaType())
	mockHTTPClient.AssertNumberOfCalls(t, "OnGetSuccess", 1)
}

func TestGetBundle_Retry(t *testing.T) {
	mockHTTPClient := &failAfterNCallsHttpClient{
		FailOnCallN:              1,
		FailOnAllSubsequentCalls: false,
	}

	c := &LiveClient{
		httpClient: mockHTTPClient,
		logger:     io.NewTestHandler(),
	}

	b, err := c.getBundle("mybundleurl")
	require.NoError(t, err)
	require.Equal(t, "application/vnd.dev.sigstore.bundle.v0.3+json", b.GetMediaType())
	mockHTTPClient.AssertNumberOfCalls(t, "OnGetFailAfterNCalls", 2)
}

func TestGetBundle_Fail(t *testing.T) {
	mockHTTPClient := &failHttpClient{}
	c := &LiveClient{
		httpClient: mockHTTPClient,
		logger:     io.NewTestHandler(),
	}

	b, err := c.getBundle("mybundleurl")
	require.Error(t, err)
	require.Nil(t, b)
	mockHTTPClient.AssertNumberOfCalls(t, "OnGetFail", 1)
}

func TestGetTrustDomain(t *testing.T) {
	fetcher := mockMetaGenerator{
		TrustDomain: "foo",
	}

	t.Run("with returned trust domain", func(t *testing.T) {
		c := LiveClient{
			githubAPI: mockAPIClient{
				OnREST: fetcher.OnREST,
			},
			logger: io.NewTestHandler(),
		}
		td, err := c.GetTrustDomain()
		require.Nil(t, err)
		require.Equal(t, "foo", td)

	})

	t.Run("with error", func(t *testing.T) {
		c := LiveClient{
			githubAPI: mockAPIClient{
				OnREST: fetcher.OnRESTError,
			},
			logger: io.NewTestHandler(),
		}
		td, err := c.GetTrustDomain()
		require.Equal(t, "", td)
		require.ErrorContains(t, err, "test error")
	})

}

func TestGetAttestationsRetries(t *testing.T) {
	getAttestationRetryInterval = 0

	fetcher := mockDataGenerator{
		NumAttestations: 5,
	}

	c := &LiveClient{
		githubAPI: mockAPIClient{
			OnRESTWithNext: fetcher.FlakyOnRESTSuccessWithNextPageHandler(),
		},
		httpClient: &mockHttpClient{},
		logger:     io.NewTestHandler(),
	}

	attestations, err := c.GetByRepoAndDigest(testRepo, testDigest, DefaultLimit)
	require.NoError(t, err)

	// assert the error path was executed; because this is a paged
	// request, it should have errored twice
	fetcher.AssertNumberOfCalls(t, "FlakyOnRESTSuccessWithNextPage:error", 2)

	// but we still successfully got the right data
	require.Equal(t, len(attestations), 10)
	bundle := (attestations)[0].Bundle
	require.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle.v0.3+json")

	// same test as above, but for GetByOwnerAndDigest:
	attestations, err = c.GetByOwnerAndDigest(testOwner, testDigest, DefaultLimit)
	require.NoError(t, err)

	// because we haven't reset the mock, we have added 2 more failed requests
	fetcher.AssertNumberOfCalls(t, "FlakyOnRESTSuccessWithNextPage:error", 4)

	require.Equal(t, len(attestations), 10)
	bundle = (attestations)[0].Bundle
	require.Equal(t, bundle.GetMediaType(), "application/vnd.dev.sigstore.bundle.v0.3+json")
}

// test total retries
func TestGetAttestationsMaxRetries(t *testing.T) {
	getAttestationRetryInterval = 0

	fetcher := mockDataGenerator{
		NumAttestations: 5,
	}

	c := &LiveClient{
		githubAPI: mockAPIClient{
			OnRESTWithNext: fetcher.OnREST500ErrorHandler(),
		},
		logger: io.NewTestHandler(),
	}

	_, err := c.GetByRepoAndDigest(testRepo, testDigest, DefaultLimit)
	require.Error(t, err)

	fetcher.AssertNumberOfCalls(t, "OnREST500Error", 4)
}
