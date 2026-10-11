package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	for _, tc := range []struct {
		current, latest string
		want            bool
	}{
		{"0.2.0", "0.3.0", true},
		{"0.2.0", "0.2.1", true},
		{"0.2.9", "0.10.0", true},
		{"0.2.0", "0.2.0", false},
		{"0.3.0", "0.2.0", false},
		{"v0.2.0", "v0.2.1", true},
		{"0.2.0-rc1", "0.2.1", true},
		{"dev", "9.9.9", false},
		{"0.2.0", "garbage", false},
		{"1.0", "1.0.1", false},
	} {
		if got := Newer(tc.current, tc.latest); got != tc.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
		}
	}
}

// releaseServer answers the latest-release endpoint and counts the requests.
func releaseServer(t *testing.T, status int, tag string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `","html_url":"https://example.test/releases/` + tag + `"}`))
	}))
	t.Cleanup(srv.Close)

	return srv, &hits
}

func TestLatestCachesTheAnswerForADay(t *testing.T) {
	srv, hits := releaseServer(t, http.StatusOK, "v0.3.0")
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	c := &Checker{Dir: t.TempDir(), APIURL: srv.URL, Now: func() time.Time { return now }}

	got, err := c.Latest(context.Background(), false)
	if err != nil || got.Version != "0.3.0" || got.URL == "" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if _, err := c.Latest(context.Background(), false); err != nil || hits.Load() != 1 {
		t.Errorf("a second call within a day should use the cache, requests = %d, err = %v", hits.Load(), err)
	}

	now = now.Add(TTL + time.Minute)
	if _, err := c.Latest(context.Background(), false); err != nil || hits.Load() != 2 {
		t.Errorf("after a day GitHub is asked again, requests = %d, err = %v", hits.Load(), err)
	}
	if _, err := c.Latest(context.Background(), true); err != nil || hits.Load() != 3 {
		t.Errorf("force should ask again, requests = %d, err = %v", hits.Load(), err)
	}
}

func TestLatestRemembersAFailureAndKeepsTheOldAnswer(t *testing.T) {
	good, _ := releaseServer(t, http.StatusOK, "v0.3.0")
	bad, hits := releaseServer(t, http.StatusInternalServerError, "")
	dir := t.TempDir()
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	if _, err := (&Checker{Dir: dir, APIURL: good.URL, Now: clock}).Latest(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	now = now.Add(TTL + time.Minute)
	failing := &Checker{Dir: dir, APIURL: bad.URL, Now: clock}
	got, err := failing.Latest(context.Background(), false)
	if err == nil || got.Version != "0.3.0" {
		t.Fatalf("got %+v, %v; want the old answer and an error", got, err)
	}
	if _, err := failing.Latest(context.Background(), false); err != nil || hits.Load() != 1 {
		t.Errorf("a failed lookup must not be retried on every call, requests = %d, err = %v", hits.Load(), err)
	}
}

func TestCachedNeverAsksTheNetwork(t *testing.T) {
	srv, hits := releaseServer(t, http.StatusOK, "v0.3.0")
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	c := &Checker{Dir: t.TempDir(), APIURL: srv.URL, Now: func() time.Time { return now }}

	if rel, stale := c.Cached(); rel.Version != "" || !stale {
		t.Errorf("an empty cache should be stale and empty, got %+v, %v", rel, stale)
	}
	if _, err := c.Latest(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if rel, stale := c.Cached(); rel.Version != "0.3.0" || stale {
		t.Errorf("got %+v, stale %v", rel, stale)
	}
	now = now.Add(TTL + time.Minute)
	if _, stale := c.Cached(); !stale {
		t.Error("a day later the cache is stale")
	}
	if hits.Load() != 1 {
		t.Errorf("Cached must not make requests, got %d", hits.Load())
	}
}
