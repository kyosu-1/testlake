package ghapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestJobStartedAt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/app/actions/runs/42/attempts/1/jobs" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("auth = %s", r.Header.Get("Authorization"))
		}
		w.Write([]byte(`{"jobs":[
			{"name":"other","started_at":"2026-07-08T09:00:00Z"},
			{"name":"unit-tests","started_at":"2026-07-08T09:05:00Z"}]}`))
	}))
	defer srv.Close()

	got, err := JobStartedAt(srv.URL, "tok", "acme/app", 42, 1, "unit-tests")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 7, 8, 9, 5, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestJobStartedAtNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"jobs":[]}`))
	}))
	defer srv.Close()
	if _, err := JobStartedAt(srv.URL, "tok", "acme/app", 42, 1, "unit-tests"); err == nil {
		t.Fatal("want error for missing job")
	}
}
