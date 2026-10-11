# Development

## Build, install, check

```
go build -o roundhouse-dev ./cmd/roundhouse             # a development build; see "Using roundhouse while developing it"
gofmt -l .
go vet ./...
GOOS=linux go vet ./... && GOOS=darwin go vet ./...
go test -race ./...
golangci-lint run        # same linter as CI; errcheck is the usual catch
govulncheck ./...
```

`go test -race` needs cgo (a C compiler); on a plain Windows setup run it in CI or WSL instead.

CI (`.github/workflows/ci.yml`) runs tests on Linux (with `-race`), Windows and macOS, golangci-lint,
govulncheck, a license check of all dependencies, the GoReleaser configuration check, a release snapshot
build and a secret scan.

## Git workflow

- One small branch and pull request per step (`feat/...`, `fix/...`, `docs/...`).
- Conventional Commits (`feat:`, `fix:`, `docs:`, `ci:`, `chore:`, `test:`); the release notes are grouped by them.
- Code, comments and commit messages are in English.

## Tests without Claude Code

- Unit tests use temp dirs and the environment variables `ROUNDHOUSE_CONFIG_DIR`, `ROUNDHOUSE_DATA_DIR` and
  `CLAUDE_CONFIG_DIR`; they never touch your real chats.
- Transcripts for tests are small synthetic files (`internal/transcript/testdata`) or written on the fly.
  Do not commit real chat content.
- The list is tested without a terminal: the tests feed `tea.Msg` values into `tui.Model` and inspect the items
  and the rendered `View()`. A width test checks that no line is wider than the window.
- Starters are tested by building the command and checking arguments and working directory; no process is started.
- To try the CLI end to end without Claude, point the config at a harmless command:

  ```toml
  starter = "probe"

  [[starters]]
  name = "probe"
  resume = ["cmd", "/c", "cd & echo resume {id}"]   # on Linux: ["sh", "-c", "pwd; echo resume {id}"]
  new = ["cmd", "/c", "cd & echo new"]
  ```

The mod has its own checks: `claude plugin validate mod` and `claude plugin test mod` (tests in `mod/tests`, run against a fake `roundhouse`), plus a strict `tsc` run over `mod/hooks`, `mod/types` and `mod/tests` with the declaration file Claude Code writes.

See [MANUAL_TESTS.md](MANUAL_TESTS.md) for the checks that need a real terminal and a real `claude`.

## Using roundhouse while developing it

Keep the roundhouse you work with every day apart from the one you build, so that neither gets in the other's way.

- **Daily use** is a release: download the archive for your system from the releases page, check it against `checksums.txt`, and put
  `roundhouse` in a fixed folder on your `PATH` (on Windows for example `%LocalAppData%\Programs\roundhouse`). `roundhouse --version`
  shows the release. Update it with `roundhouse update`, which asks first, or by downloading the next archive. Install the mod with `roundhouse mod install`, which takes it from
  GitHub, so it is the published one too.
- **Do not run `go install ./cmd/roundhouse`** on that machine: it puts a development build into `GOPATH/bin`, which then
  shadows or replaces the release.
- **Development builds** get their own name and their own data, so they do not touch the favorites and the index cache of
  the release:

  ```
  go build -o roundhouse-dev ./cmd/roundhouse                  # roundhouse-dev.exe on Windows; both are ignored by git
  export ROUNDHOUSE_DATA_DIR=/tmp/rh-dev-data ROUNDHOUSE_CONFIG_DIR=/tmp/rh-dev-config
  ./roundhouse-dev
  ```

  (PowerShell: `$env:ROUNDHOUSE_DATA_DIR = "$env:TEMP\rh-dev-data"`, and the same for `ROUNDHOUSE_CONFIG_DIR`.) The chats themselves are
  only read, from the same Claude folder as always.
- **The mod in development** loads from the checkout with `claude --plugin-dir mod`. It uses the `roundhouse` on the `PATH`, so a
  mod change that needs a new CLI feature has to wait for a release, or be tried with the development build first on the `PATH`
  of that one terminal. If the installed `roundhouse-mod` gets in the way, turn it off for the test with
  `claude plugin disable roundhouse-mod@roundhouse` and back on with `claude plugin enable roundhouse-mod@roundhouse`.
- **Releases** are made by tagging `main` (see "Releasing"); `main` itself only changes through pull requests.

## Releasing

Releases are made by pushing a tag `vX.Y.Z`. The release workflow runs GoReleaser (archives, `.deb` and `.rpm`,
checksums) and signs `checksums.txt` keyless with cosign. `main` should be green first. The first tag also
publishes a signature entry for the repository in the public Sigstore transparency log, so tag only when the
repository is meant to be public.
