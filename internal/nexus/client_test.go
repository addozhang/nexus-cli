package nexus_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/addozhang/nexus-cli/internal/nexus"
)

func newSearchServer(t *testing.T, pages map[string][]nexus.RawComponent) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/service/rest/v1/search", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("continuationToken")
		items, ok := pages[token]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp := struct {
			Items             []nexus.RawComponent `json:"items"`
			ContinuationToken string               `json:"continuationToken,omitempty"`
		}{Items: items}
		if token == "" && len(pages) > 1 {
			resp.ContinuationToken = "page2"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return httptest.NewServer(mux)
}

func Test_Search_FollowsContinuationTokens(t *testing.T) {
	srv := newSearchServer(t, map[string][]nexus.RawComponent{
		"":      {{ID: "1", Name: "a", Version: "1"}},
		"page2": {{ID: "2", Name: "b", Version: "2"}},
	})
	defer srv.Close()

	c := nexus.New(srv.URL, "", "", nil)
	got, err := c.Search(context.Background(), nexus.SearchParams{Format: "npm"}, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d components across pages, want 2", len(got))
	}
}

func Test_Search_LimitCapsResults(t *testing.T) {
	page := make([]nexus.RawComponent, 30)
	for i := range page {
		page[i] = nexus.RawComponent{ID: string(rune('a' + i)), Name: "x", Version: "1"}
	}
	srv := newSearchServer(t, map[string][]nexus.RawComponent{"": page})
	defer srv.Close()

	c := nexus.New(srv.URL, "", "", nil)
	got, err := c.Search(context.Background(), nexus.SearchParams{Format: "npm"}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 5 {
		t.Errorf("got %d components, want capped at 5", len(got))
	}
}

type apiHandler func(w http.ResponseWriter, r *http.Request)

func Test_Client_ErrorTranslation(t *testing.T) {
	tests := []struct {
		name    string
		handler apiHandler
		wantErr string
	}{
		{
			name: "401 becomes auth error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: "auth",
		},
		{
			name: "500 becomes response error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: "response",
		},
		{
			name: "malformed JSON becomes response error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("{not json"))
			},
			wantErr: "response",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/service/rest/v1/repositories", tt.handler)
			srv := httptest.NewServer(mux)
			defer srv.Close()

			c := nexus.New(srv.URL, "", "", nil)
			_, err := c.Repositories(context.Background())
			if err == nil || !contains(err.Error(), tt.wantErr) {
				t.Errorf("Repositories() error = %v, want class %q", err, tt.wantErr)
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
