// Package update finds out whether a newer roundhouse release exists and, when
// the user has agreed, installs it. Looking up the latest release only reads
// the public release list of the repository; nothing is installed without a
// call to Installer.Install, which the update command makes after asking.
package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// Repo is the GitHub repository whose releases are checked.
	Repo = "LuGG000/roundhouse"
	// ReleasesPage lists the releases for humans.
	ReleasesPage = "https://github.com/" + Repo + "/releases"

	cacheFile = "update-check.json"
	// TTL is how long a check result is reused before GitHub is asked again.
	TTL = 24 * time.Hour
)

// Release is a published version.
type Release struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// Checker looks up the latest release and caches the answer in Dir.
type Checker struct {
	// Dir holds the cache file, normally the data dir.
	Dir string
	// APIURL is the "latest release" endpoint; empty means GitHub's.
	APIURL string
	Client *http.Client
	Now    func() time.Time
}

type cache struct {
	CheckedAt time.Time `json:"checked_at"`
	Release
}

// Latest returns the newest release. A cached answer younger than TTL is used
// unless force is set. A failed lookup is cached as well, so a machine without
// network access does not retry on every command; the previous answer, if any,
// is returned together with the error.
func (c *Checker) Latest(ctx context.Context, force bool) (Release, error) {
	now := c.now()
	stored, _ := c.read()
	if !force && !stored.CheckedAt.IsZero() && now.Sub(stored.CheckedAt) < TTL {
		return stored.Release, nil
	}
	release, err := c.fetch(ctx)
	if err != nil {
		stored.CheckedAt = now
		_ = c.write(stored)
		return stored.Release, err
	}
	_ = c.write(cache{CheckedAt: now, Release: release})
	return release, nil
}

// Cached returns the last known latest release without asking GitHub, and
// whether that answer is older than TTL.
func (c *Checker) Cached() (release Release, isStale bool) {
	stored, _ := c.read()
	return stored.Release, stored.CheckedAt.IsZero() || c.now().Sub(stored.CheckedAt) >= TTL
}

func (c *Checker) fetch(ctx context.Context) (Release, error) {
	url := c.APIURL
	if url == "" {
		url = "https://api.github.com/repos/" + Repo + "/releases/latest"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("release lookup: %s", resp.Status)
	}
	var body struct {
		Tag string `json:"tag_name"`
		URL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Release{}, fmt.Errorf("release lookup: %w", err)
	}
	if body.Tag == "" {
		return Release{}, errors.New("release lookup: no tag")
	}
	return Release{Version: strings.TrimPrefix(body.Tag, "v"), URL: body.URL}, nil
}

func (c *Checker) read() (cache, error) {
	var stored cache
	data, err := os.ReadFile(filepath.Join(c.Dir, cacheFile))
	if errors.Is(err, fs.ErrNotExist) {
		return stored, nil
	}
	if err != nil {
		return stored, err
	}
	err = json.Unmarshal(data, &stored)
	return stored, err
}

func (c *Checker) write(stored cache) error {
	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(c.Dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.Dir, cacheFile), data, 0o600)
}

func (c *Checker) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// Newer reports whether latest is a higher release than current. Versions are
// compared as major.minor.patch; a pre-release suffix is ignored, and a version
// that does not parse (a development build) is never considered outdated.
func Newer(current, latest string) bool {
	cur, ok := parse(current)
	if !ok {
		return false
	}
	lat, ok := parse(latest)
	if !ok {
		return false
	}
	for i := range cur {
		if lat[i] != cur[i] {
			return lat[i] > cur[i]
		}
	}
	return false
}

func parse(version string) ([3]int, bool) {
	var parts [3]int
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if i := strings.IndexAny(version, "-+"); i >= 0 {
		version = version[:i]
	}
	fields := strings.Split(version, ".")
	if len(fields) != 3 {
		return parts, false
	}
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil || n < 0 {
			return parts, false
		}
		parts[i] = n
	}
	return parts, true
}
