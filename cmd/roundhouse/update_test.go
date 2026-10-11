package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LuGG000/roundhouse/internal/config"
	"github.com/LuGG000/roundhouse/internal/update"
)

type fakeLooker struct {
	rel   update.Release
	err   error
	force bool
}

func (f *fakeLooker) Latest(_ context.Context, force bool) (update.Release, error) {
	f.force = force
	return f.rel, f.err
}

type fakeInstaller struct {
	err     error
	version string
	target  string
	calls   int
}

func (f *fakeInstaller) Install(_ context.Context, version, target string) error {
	f.calls++
	f.version, f.target = version, target
	return f.err
}

func runFlow(t *testing.T, d updateDeps) (string, error) {
	t.Helper()
	var out bytes.Buffer
	d.out = &out
	if d.in == nil {
		d.in = strings.NewReader("")
	}
	err := updateFlow(context.Background(), d)
	return out.String(), err
}

func newer() *fakeLooker {
	return &fakeLooker{rel: update.Release{Version: "0.3.0", URL: "https://example.test/v0.3.0"}}
}

func deps(l *fakeLooker, in *fakeInstaller, answer string) updateDeps {
	return updateDeps{checker: l, installer: in, current: "0.2.0", exe: "/bin/roundhouse", in: strings.NewReader(answer), interactive: true}
}

func TestUpdateInstallsOnlyAfterAYes(t *testing.T) {
	for _, answer := range []string{"y\n", "yes\n", "YES\n", " Y \n"} {
		inst := &fakeInstaller{}
		out, err := runFlow(t, deps(newer(), inst, answer))
		if err != nil || inst.calls != 1 {
			t.Fatalf("answer %q: err %v, installs %d", answer, err, inst.calls)
		}
		if inst.version != "0.3.0" || inst.target != "/bin/roundhouse" {
			t.Errorf("installed %q over %q", inst.version, inst.target)
		}
		for _, want := range []string{"0.3.0 is available", "https://example.test/v0.3.0", "/bin/roundhouse", "Install roundhouse 0.3.0? [y/N]", "Updated to roundhouse 0.3.0"} {
			if !strings.Contains(out, want) {
				t.Errorf("answer %q: output misses %q:\n%s", answer, want, out)
			}
		}
	}
}

func TestUpdateChangesNothingWithoutAYes(t *testing.T) {
	for _, answer := range []string{"\n", "n\n", "no\n", "maybe\n", "", "yess\n", "y es\n"} {
		inst := &fakeInstaller{}
		out, err := runFlow(t, deps(newer(), inst, answer))
		if err != nil || inst.calls != 0 {
			t.Fatalf("answer %q: err %v, installs %d; want no install", answer, err, inst.calls)
		}
		if !strings.Contains(out, "Nothing was changed.") {
			t.Errorf("answer %q: output = %q", answer, out)
		}
	}
}

func TestUpdateRefusesWithoutATerminalToAnswerIn(t *testing.T) {
	inst := &fakeInstaller{}
	d := deps(newer(), inst, "y\n") // the answer is there, but nobody is asked
	d.interactive = false
	_, err := runFlow(t, d)
	if err == nil || !strings.Contains(err.Error(), "needs a terminal") || inst.calls != 0 {
		t.Errorf("err = %v, installs = %d; want a refusal and no install", err, inst.calls)
	}
}

func TestUpdateInstallsTheShownVersionAndReportsAFailure(t *testing.T) {
	inst := &fakeInstaller{err: errors.New("checksum mismatch")}
	_, err := runFlow(t, deps(newer(), inst, "y\n"))
	if err == nil || !strings.Contains(err.Error(), "nothing was changed") || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("err = %v", err)
	}
}

func TestUpdateAsksGitHubAndSaysWhenYouAreCurrent(t *testing.T) {
	l := &fakeLooker{rel: update.Release{Version: "0.2.0"}}
	inst := &fakeInstaller{}
	out, err := runFlow(t, deps(l, inst, "y\n"))
	if err != nil || inst.calls != 0 || !l.force || !strings.Contains(out, "up to date") {
		t.Errorf("err %v, installs %d, force %v, out %q", err, inst.calls, l.force, out)
	}
}

func TestUpdateReportsAFailedLookup(t *testing.T) {
	inst := &fakeInstaller{}
	_, err := runFlow(t, deps(&fakeLooker{err: errors.New("offline")}, inst, "y\n"))
	if err == nil || !strings.Contains(err.Error(), "offline") || inst.calls != 0 {
		t.Errorf("err = %v, installs = %d", err, inst.calls)
	}
}

func TestDevelopmentBuildsHaveNothingToUpdateTo(t *testing.T) {
	l, inst := newer(), &fakeInstaller{}
	d := deps(l, inst, "y\n")
	d.current = devVersion
	out, err := runFlow(t, d)
	if err != nil || inst.calls != 0 || !strings.Contains(out, "development build") {
		t.Errorf("err %v, installs %d, out %q", err, inst.calls, out)
	}
}

func TestCheckOnlyLooksAndNeverAsks(t *testing.T) {
	inst := &fakeInstaller{}
	d := deps(newer(), inst, "y\n")
	d.check = true
	out, err := runFlow(t, d)
	if err != nil || inst.calls != 0 {
		t.Fatalf("err %v, installs %d", err, inst.calls)
	}
	if strings.Contains(out, "[y/N]") || !strings.Contains(out, "0.3.0 is available (you have 0.2.0)") {
		t.Errorf("out = %q", out)
	}
}

func TestCheckJSON(t *testing.T) {
	d := deps(newer(), &fakeInstaller{}, "")
	d.check, d.asJSON = true, true
	out, err := runFlow(t, d)
	if err != nil {
		t.Fatal(err)
	}
	var got checkJSON
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	want := checkJSON{Schema: 1, Current: "0.2.0", Latest: "0.3.0", UpdateAvailable: true, URL: "https://example.test/v0.3.0"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}

	d = deps(&fakeLooker{rel: update.Release{Version: "0.2.0"}}, &fakeInstaller{}, "")
	d.check, d.asJSON = true, true
	out, _ = runFlow(t, d)
	if err := json.Unmarshal([]byte(out), &got); err != nil || got.UpdateAvailable {
		t.Errorf("an equal version is no update: %+v, %v", got, err)
	}
}

// cachedRelease fills the update cache of the data dir with a release v<tag>.
func cachedRelease(t *testing.T, tag string) {
	t.Helper()
	t.Setenv("ROUNDHOUSE_DATA_DIR", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v` + tag + `","html_url":"https://example.test/` + tag + `"}`))
	}))
	t.Cleanup(srv.Close)
	checker, err := newChecker()
	if err != nil {
		t.Fatal(err)
	}
	checker.APIURL = srv.URL
	if _, err := checker.Latest(context.Background(), true); err != nil {
		t.Fatal(err)
	}
}

func withVersion(t *testing.T, v string) {
	t.Helper()
	old := version
	version = v
	t.Cleanup(func() { version = old })
}

func TestAvailableUpdateReadsTheCacheWithoutTheNetwork(t *testing.T) {
	cachedRelease(t, "0.3.0")
	withVersion(t, "0.2.0")
	e := &env{cfg: config.Default()}

	rel, ok := e.availableUpdate(context.Background(), false)
	if !ok || rel.Version != "0.3.0" || rel.URL == "" {
		t.Errorf("got %+v, %v; want 0.3.0", rel, ok)
	}

	withVersion(t, "0.3.0")
	if _, ok := e.availableUpdate(context.Background(), false); ok {
		t.Error("the current version is no update")
	}
	withVersion(t, "0.4.0")
	if _, ok := e.availableUpdate(context.Background(), false); ok {
		t.Error("a newer build than the release is no update")
	}
}

func TestAvailableUpdateIsOffForDevBuildsAndWhenSwitchedOff(t *testing.T) {
	cachedRelease(t, "9.9.9")

	withVersion(t, devVersion)
	if _, ok := (&env{cfg: config.Default()}).availableUpdate(context.Background(), false); ok {
		t.Error("a development build must not report an update")
	}

	withVersion(t, "0.2.0")
	off := config.Default()
	off.UpdateCheck = false
	if _, ok := (&env{cfg: off}).availableUpdate(context.Background(), false); ok {
		t.Error("update_check = false must switch the notice off")
	}
}
