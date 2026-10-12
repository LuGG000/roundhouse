package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// maxDownload bounds every downloaded file.
const maxDownload = 100 << 20

// Installer replaces the roundhouse binary with a release the user chose. It
// does not touch running processes: a running binary keeps working (on Windows
// it is renamed aside, elsewhere the new file is renamed over it), and
// everything started afterwards uses the new binary.
type Installer struct {
	// APIBase is the release API of the repository; empty means GitHub's.
	APIBase string
	// AssetPrefix is the only place files are downloaded from; empty means
	// the release downloads of Repo on github.com.
	AssetPrefix string
	Client      *http.Client
	// GOOS and GOARCH select the archive; empty means this build's.
	GOOS, GOARCH string
	// Cosign is the cosign program that verifies the signature of checksums.txt. Empty means none is
	// installed: the archive is then checked against the checksums only, and Notice says so.
	Cosign string
	// Run runs cosign; nil runs it for real.
	Run func(*exec.Cmd) error
	// Notice receives what the user should know about the check; nil drops it.
	Notice func(string)
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// Install downloads exactly the given version, checks the archive against the
// checksums published with that release and puts the binary at target. It
// installs nothing when anything does not match.
func (in *Installer) Install(ctx context.Context, version, target string) error {
	version = strings.TrimPrefix(version, "v")
	assets, err := in.assets(ctx, version)
	if err != nil {
		return err
	}
	goos, goarch := in.GOOS, in.GOARCH
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	name := fmt.Sprintf("roundhouse_%s_%s_%s.%s", version, goos, goarch, ext)
	archive, ok := assets[name]
	if !ok {
		return fmt.Errorf("release %s has no %s", version, name)
	}
	sums, ok := assets["checksums.txt"]
	if !ok {
		return fmt.Errorf("release %s has no checksums.txt", version)
	}

	sumData, err := in.get(ctx, sums, "application/octet-stream", true)
	if err != nil {
		return err
	}
	if err := in.verifySignature(ctx, assets, sumData); err != nil {
		return err
	}
	want, err := checksumFor(sumData, name)
	if err != nil {
		return err
	}
	data, err := in.get(ctx, archive, "application/octet-stream", true)
	if err != nil {
		return err
	}
	if got := sha256.Sum256(data); hex.EncodeToString(got[:]) != want {
		return fmt.Errorf("%s does not match its checksum; nothing was installed", name)
	}
	binary := "roundhouse"
	if goos == "windows" {
		binary += ".exe"
	}
	exe, err := extract(data, ext, binary)
	if err != nil {
		return err
	}

	return Replace(target, exe)
}

// assets lists the files of the release for version by name.
func (in *Installer) assets(ctx context.Context, version string) (map[string]string, error) {
	base := in.APIBase
	if base == "" {
		base = "https://api.github.com/repos/" + Repo
	}
	data, err := in.get(ctx, base+"/releases/tags/v"+version, "application/vnd.github+json", false)
	if err != nil {
		return nil, err
	}
	var body struct {
		Tag    string  `json:"tag_name"`
		Assets []asset `json:"assets"`
	}
	if err := json.Unmarshal(data, &body); err != nil || body.Tag != "v"+version {
		return nil, errors.New("release lookup: unexpected answer")
	}
	assets := make(map[string]string, len(body.Assets))
	for _, a := range body.Assets {
		assets[a.Name] = a.URL
	}

	return assets, nil
}

// get fetches url. A download (isAsset) must come from the release downloads of
// this repository, whatever the release answer claims.
func (in *Installer) get(ctx context.Context, url, accept string, isAsset bool) ([]byte, error) {
	if isAsset {
		prefix := in.AssetPrefix
		if prefix == "" {
			prefix = "https://github.com/" + Repo + "/releases/download/"
		}
		if !strings.HasPrefix(url, prefix) {
			return nil, fmt.Errorf("download: %s is not a release file of this repository", url)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	client := in.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDownload {
		return nil, errors.New("download: file is too large")
	}

	return data, nil
}

// checksumFor finds the sha256 of name in a checksums.txt ("<hex>  <name>" per line).
func checksumFor(sums []byte, name string) (string, error) {
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			return strings.ToLower(fields[0]), nil
		}
	}

	return "", fmt.Errorf("checksums.txt has no entry for %s", name)
}

// extract returns the file called name from a tar.gz or zip archive.
func extract(data []byte, ext, name string) ([]byte, error) {
	if ext == "zip" {
		r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, err
		}
		for _, f := range r.File {
			if f.Name != name {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer func() { _ = rc.Close() }()

			return readLimited(rc)
		}

		return nil, fmt.Errorf("archive has no %s", name)
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("archive has no %s", name)
		}
		if err != nil {
			return nil, err
		}
		if hdr.Typeflag == tar.TypeReg && hdr.Name == name {
			return readLimited(tr)
		}
	}
}

func readLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxDownload+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDownload {
		return nil, errors.New("archive: file is too large")
	}

	return data, nil
}

// OldSuffix marks the previous binary that Replace leaves next to the new one
// on Windows, where a running executable can be renamed but not deleted.
const OldSuffix = ".old"

// Replace puts exe at target. On Windows the running file is renamed aside
// first; CleanOld removes it at a later start.
func Replace(target string, exe []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, filepath.Base(target)+".new-*")
	if err != nil {
		return fmt.Errorf("%w (is %s writable? install it again from the releases page)", err, dir)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(exe); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		old := freeOldName(target)
		if err := os.Rename(target, old); err != nil {
			return err
		}
		if err := os.Rename(tmp.Name(), target); err != nil {
			_ = os.Rename(old, target)
			return err
		}

		return nil
	}

	return os.Rename(tmp.Name(), target)
}

// CleanOld removes the binary left behind by a Windows update. A failure is
// not reported: the file only takes up disk space.
func CleanOld(target string) {
	for _, name := range oldNames(target) {
		_ = os.Remove(name)
	}
}

// oldNames are the places Replace moves a binary aside to: .old, and .old1 to .old9 for when an earlier
// one is still being run and cannot be deleted.
func oldNames(target string) []string {
	names := []string{target + OldSuffix}
	for i := 1; i < 10; i++ {
		names = append(names, target+OldSuffix+strconv.Itoa(i))
	}

	return names
}

// freeOldName is the first of the old names that is not taken; it deletes a leftover to make one free.
func freeOldName(target string) string {
	for _, name := range oldNames(target) {
		if err := os.Remove(name); err == nil || errors.Is(err, os.ErrNotExist) {
			return name
		}
	}

	return target + OldSuffix
}

// signedBy is who must have signed checksums.txt: the release workflow of this repository, on a tag.
var signedBy = "^https://github.com/" + regexp.QuoteMeta(Repo) + `/\.github/workflows/release\.yml@refs/tags/v.*$`

// verifySignature checks the cosign signature of the checksums with the installed cosign. Without
// cosign it only says that the signature was not checked; with cosign a missing or wrong signature
// stops the update.
func (in *Installer) verifySignature(ctx context.Context, assets map[string]string, sums []byte) error {
	if in.Cosign == "" {
		in.notice("The signature of checksums.txt was not checked (cosign is not installed).")
		return nil
	}
	url, ok := assets["checksums.txt.sigstore.json"]
	if !ok {
		return errors.New("release has no checksums.txt.sigstore.json to check with cosign")
	}
	bundle, err := in.get(ctx, url, "application/octet-stream", true)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "roundhouse-verify-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	sumsFile, bundleFile := filepath.Join(dir, "checksums.txt"), filepath.Join(dir, "checksums.txt.sigstore.json")
	if err := os.WriteFile(sumsFile, sums, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(bundleFile, bundle, 0o600); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, in.Cosign, "verify-blob", sumsFile, "--bundle", bundleFile,
		"--certificate-identity-regexp", signedBy,
		"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com")
	run := in.Run
	if run == nil {
		run = func(c *exec.Cmd) error {
			if out, err := c.CombinedOutput(); err != nil {
				return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
			}
			return nil
		}
	}
	if err := run(cmd); err != nil {
		return fmt.Errorf("cosign could not verify checksums.txt: %w", err)
	}
	in.notice("The signature of checksums.txt was verified with cosign.")

	return nil
}

func (in *Installer) notice(text string) {
	if in.Notice != nil {
		in.Notice(text)
	}
}
