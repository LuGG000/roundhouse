# Development

## Build, install, check

```
go build -o ~/.local/bin/roundhouse ./cmd/roundhouse   # rebuild after every pull
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

See [MANUAL_TESTS.md](MANUAL_TESTS.md) for the checks that need a real terminal and a real `claude`.

## Releasing

Releases are made by pushing a tag `vX.Y.Z`. The release workflow runs GoReleaser (archives, `.deb` and `.rpm`,
checksums) and signs `checksums.txt` keyless with cosign. `main` should be green first. The first tag also
publishes a signature entry for the repository in the public Sigstore transparency log, so tag only when the
repository is meant to be public.
