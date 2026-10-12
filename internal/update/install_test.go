package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
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
	bundle   bool // the release carries checksums.txt.sigstore.json
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
			`{"name":"checksums.txt","browser_download_url":"` + r.srv.URL + `/download/checksums.txt"}` + bundleAsset(r) + `]}`))
	})
	mux.HandleFunc("/download/checksums.txt.sigstore.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("bundle"))
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

func bundleAsset(r *release) string {
	if !r.bundle {
		return ""
	}

	return `,{"name":"checksums.txt.sigstore.json","browser_download_url":"` + r.srv.URL + `/download/checksums.txt.sigstore.json"}`
}

// signed installs a release with a signature bundle through a cosign that is run by run.
func signed(t *testing.T, run func(*exec.Cmd) error) (string, []string, error) {
	t.Helper()
	r := serve(t, "linux", "amd64", tarGzOf(t, "roundhouse", []byte("new")), func(r *release) { r.bundle = true })
	in := r.installer("linux", "amd64")
	var notices []string
	in.Cosign, in.Run, in.Notice = "cosign", run, func(s string) { notices = append(notices, s) }
	dst := target(t)
	err := in.Install(context.Background(), "0.3.0", dst)
	got, _ := os.ReadFile(dst)

	return string(got), notices, err
}

func TestInstallVerifiesTheSignatureWithCosign(t *testing.T) {
	var args []string
	got, notices, err := signed(t, func(c *exec.Cmd) error {
		args = c.Args
		b, readErr := os.ReadFile(c.Args[2]) // the checksums file, read while it still exists
		if readErr != nil || !strings.Contains(string(b), "roundhouse_0.3.0_linux_amd64.tar.gz") {
			t.Errorf("cosign got no checksums file: %v", readErr)
		}
		return nil
	})
	if err != nil || got != "new" {
		t.Fatalf("err = %v, binary = %q", err, got)
	}
	line := strings.Join(args, " ")
	for _, want := range []string{"verify-blob", "--bundle", "--certificate-identity-regexp ^https://github.com/LuGG000/roundhouse/", "--certificate-oidc-issuer https://token.actions.githubusercontent.com"} {
		if !strings.Contains(line, want) {
			t.Errorf("cosign line misses %q: %s", want, line)
		}
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "verified") {
		t.Errorf("notices = %q", notices)
	}
}

func TestInstallStopsWhenCosignRejectsTheSignature(t *testing.T) {
	got, _, err := signed(t, func(*exec.Cmd) error { return errors.New("no matching signature") })
	if err == nil || !strings.Contains(err.Error(), "no matching signature") {
		t.Fatalf("err = %v", err)
	}
	if got != "old" {
		t.Errorf("binary = %q, want it untouched", got)
	}
}

func TestInstallWithCosignNeedsTheSignatureBundle(t *testing.T) {
	r := serve(t, "linux", "amd64", tarGzOf(t, "roundhouse", []byte("new")), nil)
	in := r.installer("linux", "amd64")
	in.Cosign, in.Run = "cosign", func(*exec.Cmd) error { t.Error("cosign ran"); return nil }
	dst := target(t)
	if err := in.Install(context.Background(), "0.3.0", dst); err == nil || !strings.Contains(err.Error(), "sigstore.json") {
		t.Fatalf("err = %v", err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "old" {
		t.Errorf("binary = %q, want it untouched", got)
	}
}

func TestInstallWithoutCosignSaysTheSignatureWasNotChecked(t *testing.T) {
	r := serve(t, "linux", "amd64", tarGzOf(t, "roundhouse", []byte("new")), nil)
	in := r.installer("linux", "amd64")
	var notices []string
	in.Notice = func(s string) { notices = append(notices, s) }
	if err := in.Install(context.Background(), "0.3.0", target(t)); err != nil {
		t.Fatal(err)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], "not checked") {
		t.Errorf("notices = %q", notices)
	}
}

func TestReplaceMovesAsideWhenTheOldBinaryIsStillRunning(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only Windows moves the running binary aside")
	}
	path := target(t)
	if err := os.WriteFile(path+OldSuffix, []byte("older"), 0o755); err != nil {
		t.Fatal(err)
	}
	running, err := os.Open(path + OldSuffix) // an open file cannot be deleted, like a running program
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = running.Close() }()

	if err := Replace(path, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); got != "new" {
		t.Errorf("binary = %q", got)
	}
	if got := read(t, path+OldSuffix+"1"); got != "old" {
		t.Errorf("moved aside = %q", got)
	}
}
