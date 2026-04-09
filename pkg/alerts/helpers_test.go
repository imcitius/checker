// SPDX-License-Identifier: BUSL-1.1

package alerts

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostJSON_Success(t *testing.T) {
	var receivedContentType string
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	payload := map[string]string{"key": "value"}
	headers := map[string]string{"Authorization": "Bearer token123"}
	err := postJSON(srv.URL, payload, headers)
	require.NoError(t, err)
	assert.Equal(t, "application/json", receivedContentType)
	assert.Equal(t, "Bearer token123", receivedAuth)
}

func TestPostJSON_NilHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := postJSON(srv.URL, map[string]string{"a": "b"}, nil)
	assert.NoError(t, err)
}

func TestPostJSON_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	err := postJSON(srv.URL, map[string]string{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 500")
}

func TestPostJSON_ClientError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer srv.Close()

	err := postJSON(srv.URL, map[string]string{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 400")
}

func TestPostJSON_InvalidURL(t *testing.T) {
	err := postJSON("http://127.0.0.1:1/nonexistent", map[string]string{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sending request")
}

func TestPostJSON_UnmarshalablePayload(t *testing.T) {
	// A channel cannot be marshaled to JSON.
	err := postJSON("http://localhost", make(chan int), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshaling payload")
}

func TestPostJSONWithClient_CustomClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{}
	err := postJSONWithClient(client, srv.URL, map[string]string{"k": "v"}, nil)
	assert.NoError(t, err)
}

func TestPostJSON_MultipleHeaders(t *testing.T) {
	var gotHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	headers := map[string]string{
		"X-Custom-One": "val1",
		"X-Custom-Two": "val2",
	}
	err := postJSON(srv.URL, map[string]string{}, headers)
	require.NoError(t, err)
	assert.Equal(t, "val1", gotHeaders.Get("X-Custom-One"))
	assert.Equal(t, "val2", gotHeaders.Get("X-Custom-Two"))
}

func TestPostJSON_SuccessRange(t *testing.T) {
	// 201 Created should be treated as success.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	err := postJSON(srv.URL, map[string]string{}, nil)
	assert.NoError(t, err)
}

func TestDefaultHTTPClientTimeout(t *testing.T) {
	assert.NotNil(t, defaultHTTPClient)
	assert.NotNil(t, insecureHTTPClient)
}
