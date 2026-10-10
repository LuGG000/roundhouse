# Contributing

- Only permissive dependencies (MIT, BSD, Apache-2.0, ISC); check the license before adding one.
- Never write to Claude Code's files (transcripts, settings); roundhouse only reads them.
- Never read, copy or log credential files or tokens.
- Be respectful; see [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).
- Keep changes small and match the surrounding style. Code, comments and commits are in English; commits follow Conventional Commits.
- `gofmt -l .`, `go vet ./...` and `go test ./...` must pass; CI also runs golangci-lint, govulncheck and a race-enabled test run.
- Do not put real chat content into tests or fixtures; write small synthetic transcripts.
- See `docs/DEVELOPMENT.md` for build, test and pull request workflow.
