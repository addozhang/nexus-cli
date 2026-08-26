package nexus_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
	"github.com/addozhang/nexus-cli/internal/nexus"
)

func Test_Status_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := nexus.New(srv.URL, "u", "t", nil).Status(context.Background()); err != nil {
		t.Fatalf("Status: %v", err)
	}
}

func Test_Status_NotFoundIsResponseError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	err := nexus.New(srv.URL, "", "", nil).Status(context.Background())
	var nxErr *nxerrors.Error
	if !errors.As(err, &nxErr) || nxErr.Class() != nxerrors.ClassResponse {
		t.Fatalf("want ClassResponse, got %v", err)
	}
}

func Test_GetComponent(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/service/rest/v1/components/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"c1","name":"lib","version":"1.0","repository":"r"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := nexus.New(srv.URL, "", "", nil).GetComponent(context.Background(), "c1")
	if err != nil || c.Name != "lib" || c.Version != "1.0" {
		t.Fatalf("GetComponent = %+v, %v", c, err)
	}
}

func Test_TLSErrorClassification(t *testing.T) {
	// Server with TLS; client without proper roots → certificate failure.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()

	c := nexus.New(srv.URL, "", "", nil) // default transport rejects the test cert
	_, err := c.Repositories(context.Background())
	var nxErr *nxerrors.Error
	if !errors.As(err, &nxErr) || nxErr.Class() != nxerrors.ClassTLS {
		t.Fatalf("want ClassTLS for bad cert, got %v", err)
	}
}
