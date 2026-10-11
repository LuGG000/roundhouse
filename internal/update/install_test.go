package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func zipOf(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func tarGzOf(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

// release serves one release v0.3.0 with an archive for goos/goarch and its checksums.
type release struct {
	srv      *httptest.Server
	archive  []byte
	checksum string
	tag      string
}

func serve(t *testing.T, goos, goarch string, archive []byte, mutate func(*release)) *release {
	t.Helper()
	r := &release{archive: archive, tag: "v0.3.0"}
	sum := sha256.Sum256(archive)
	r.checksum = hex.EncodeToString(sum[:])
	if mutate != nil {
		mutate(r)
	}
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	name := "roundhouse_0.3.0_" + goos + "_" + goarch + "." + ext

	mux := http.NewServeMux()
	r.srv = httptest.NewServer(mux)
	t.Cleanup(r.srv.Close)
	mux.HandleFunc("/repos/x/releases/tags/v0.3.0", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"` + r.tag + `","assets":[` +
			`{"name":"` + name + `","browser_download_url":"` + r.srv.URL + `/download/` + name + `"},` +
			`{"name":"checksums.txt","browser_download_url":"` + r.srv.URL + `/download/checksums.txt"}]}`))
	})
	mux.HandleFunc("/download/checksums.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(r.checksum + "  " + name + "\n"))
	})
	mux.HandleFunc("/download/"+name, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(r.archive)
	})

	return r
}

func (r *release) installer(goos, goarch string) *Installer {
	return &Installer{APIBase: r.srv.URL + "/repos/x", AssetPrefix: r.srv.URL + "/download/", GOOS: goos, GOARCH: goarch}
}

func target(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "roundhouse")
	if err := os.WriteFile(path, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

func TestInstallReplacesTheBinaryFromAZip(t *testing.T) {
	r := serve(t, "windows", "amd64", zipOf(t, "roundhouse.exe", []byte("new")), nil)
	path := target(t)
	if err := r.installer("windows", "amd64").Install(context.Background(), "0.3.0", path); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != "new" {
		t.Errorf("binary = %q, want new", got)
	}
	if runtime.GOOS == "windows" {
		if got := read(t, path+OldSuffix); got != "old" {
			t.Errorf("the old binary should stay beside it on Windows, got %q", got)
		}
		CleanOld(path)
		if _, err := os.Stat(path + OldSuffix); err == nil {
			t.Error("CleanOld should remove the old binary")
		}
	}
}

func TestInstallFromATarGzAcceptsAVPrefix(t *testing.T) {
	r := serve(t, "linux", "arm64", tarGzOf(t, "roundhouse", []byte("new")), nil)
	path := target(t)
	if err := r.installer("linux", "arm64").Install(context.Background(), "v0.3.0", path); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != "new" {
		t.Errorf("binary = %q, want new", got)
	}
}

func TestInstallRefusesAWrongChecksumAndChangesNothing(t *testing.T) {
	r := serve(t, "linux", "amd64", tarGzOf(t, "roundhouse", []byte("new")), func(r *release) {
		r.checksum = strings.Repeat("0", 64)
	})
	path := target(t)
	err := r.installer("linux", "amd64").Install(context.Background(), "0.3.0", path)
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("err = %v, want a checksum error", err)
	}
	if got := read(t, path); got != "old" {
		t.Errorf("the binary must stay untouched, got %q", got)
	}
}

func TestInstallRefusesFilesFromAnotherPlace(t *testing.T) {
	r := serve(t, "linux", "amd64", tarGzOf(t, "roundhouse", []byte("new")), nil)
	in := r.installer("linux", "amd64")
	in.AssetPrefix = "https://github.com/someone-else/roundhouse/releases/download/"
	path := target(t)
	err := in.Install(context.Background(), "0.3.0", path)
	if err == nil || !strings.Contains(err.Error(), "not a release file") {
		t.Fatalf("err = %v, want a refusal of the download place", err)
	}
	if got := read(t, path); got != "old" {
		t.Errorf("the binary must stay untouched, got %q", got)
	}
}

func TestInstallRefusesAnAnswerForAnotherVersion(t *testing.T) {
	r := serve(t, "linux", "amd64", tarGzOf(t, "roundhouse", []byte("new")), func(r *release) { r.tag = "v9.9.9" })
	if err := r.installer("linux", "amd64").Install(context.Background(), "0.3.0", target(t)); err == nil {
		t.Error("want an error when the release answer names another version")
	}
}

func TestInstallNeedsAnArchiveForThisSystem(t *testing.T) {
	r := serve(t, "linux", "amd64", tarGzOf(t, "roundhouse", []byte("new")), nil)
	err := r.installer("linux", "riscv64").Install(context.Background(), "0.3.0", target(t))
	if err == nil || !strings.Contains(err.Error(), "has no roundhouse_0.3.0_linux_riscv64.tar.gz") {
		t.Errorf("err = %v", err)
	}
}

func TestInstallNeedsTheBinaryInsideTheArchive(t *testing.T) {
	r := serve(t, "linux", "amd64", tarGzOf(t, "something-else", []byte("new")), nil)
	path := target(t)
	if err := r.installer("linux", "amd64").Install(context.Background(), "0.3.0", path); err == nil {
		t.Error("want an error when the archive lacks the binary")
	}
	if got := read(t, path); got != "old" {
		t.Errorf("the binary must stay untouched, got %q", got)
	}
}
