# Notes for agents and new contributors

Read this first, then `README.md` and `docs/`. Open work is in the GitHub issues of this repository, not in this file.

## What this is

roundhouse is a hub for Claude Code: one list of all chats grouped by folder, to start a new chat or return to an old one in any
folder (Go, bubbletea). It reads the transcripts Claude Code keeps under `~/.claude/projects` and never writes to them. A separate
optional Claude Code mod (`mod/`, TypeScript, in-process hooks) shows the chats of the current folder in a pane. It is built to work
with [Switchyard](https://github.com/LuGG000/switchyard), the maintainer's account switcher, through a `switchyard` starter.

| Path | What |
|---|---|
| `cmd/roundhouse` | cobra commands: the list, `list`, `open`, `new`, `mod`, `update` |
| `internal/transcript`, `internal/index` | read one transcript; scan all, with a cache |
| `internal/favorites`, `internal/config`, `internal/paths` | starred chats and folders; `config.toml`; where things live |
| `internal/starter`, `internal/terminal` | the command that launches Claude Code; opening a new terminal window |
| `internal/update` | latest-release check (cached a day) and install of a chosen version |
| `internal/tui` | the bubbletea model: items, layout, keys, mouse |
| `mod/` | the mod; it only talks to the `roundhouse` command |
| `docs/` | `ARCHITECTURE`, `DEVELOPMENT`, `SETUP`, `SPIKE` (what was verified against Claude Code), `MANUAL_TESTS`, `TROUBLESHOOTING` |

`internal/` is intentional: roundhouse is an application, and these packages are not a public API. Do not move them to `pkg/`.

## Rules that are decisions, not defaults

- **Never modify Switchyard** or its repository from here. It is maintained in parallel by the same person; read its public repo only. roundhouse
  must not read Switchyard's files, settings or credentials; the only coupling is the `switchyard` starter command line.
- **The mod only talks to the CLI** (`list --json --folder`, `open --terminal`, `update --terminal`). Keep the `list --json` schema at 1 and add
  fields only; the mod reads `schema`, `version`, `update` and `sessions`. No `holdToasts`, no status line, no `--settings` in the mod, so it can sit
  beside switchyard-mod. Colors are the mod's own, kept in its `$.store`.
- **No update is ever installed without an explicit yes.** `roundhouse update` asks, has no `--yes`, refuses without a terminal, and installs exactly
  the shown version after checking the SHA-256 of the archive. The mod only opens a window that asks. The daily check is the only network request;
  `update_check = false` turns it off. Keep README, SECURITY and ARCHITECTURE true to that.
- **Never write to Claude Code's files** (transcripts, settings) and never read credentials. CI rejects tracked credential files and code that names
  `credentials.json`.
- Only permissive dependencies (MIT, BSD, Apache-2.0, ISC); `THIRD_PARTY_NOTICES.md` is regenerated with go-licenses when they change (command inside it).
- Do not commit real chat content or anything personal into tests, fixtures or docs; fixtures are small synthetic transcripts.

## Facts about Claude Code that the code relies on (details in `docs/SPIKE.md`)

- A chat is `~/.claude/projects/<project>/<id>.jsonl` (`CLAUDE_CONFIG_DIR` moves it); nested folders such as `subagents/` are not chats.
- The folder of a chat is the **first** `cwd` in the file. A chat resumed from another folder gets later lines with the other folder.
- Titles: the last `ai-title` line, else the last `last-prompt`. The format is undocumented, so unknown lines are ignored.
- `claude --resume <id>` works from other folders too, but roundhouse always starts in the original folder.
- A mod cannot restart Claude Code or change the folder of a running session, which is why the pane shows one folder.

## Working in this repository

- `main` is protected: nine required checks, linear history, no force push, also for admins. Work on a branch, open a pull request, merge with
  **rebase** (a squash merge makes GitHub use the profile name as author). Conventional Commits; code, comments, docs and commits in English.
  The maintainer talks to agents in German.
- Git identity: before the first commit set `user.name` and `user.email` locally to the maintainer's GitHub noreply identity (`LuGG000`,
  `81622672+LuGG000@users.noreply.github.com`). Never use or write a real email address. Check
  `git log --format='%an <%ae> | %cn <%ce>'` before every push. No AI attribution trailers in commits or pull requests.
  GitHub sets the committer name of server-side merges from the profile name; that is why the profile name should read `LuGG000`.
- Checks (all must pass, CI runs them): `gofmt -l .`, `go vet ./...`, `go test ./...` (`-race` runs in CI on Linux), `golangci-lint run`, `govulncheck ./...`.
  For the mod: `claude plugin validate mod`, `claude plugin test mod`, and a strict `tsc` over `mod/hooks`, `mod/types`, `mod/tests` with the declaration
  file Claude Code writes (`mod/.claude-plugin/types`, git-ignored; created when a session loads the mod with `--plugin-dir`).
- **Do not `go install ./cmd/roundhouse` on the maintainer's machine.** Their daily `roundhouse` is a release in `%LocalAppData%\Programs\roundhouse`
  (Windows). Build development versions as `roundhouse-dev` with `ROUNDHOUSE_DATA_DIR` and `ROUNDHOUSE_CONFIG_DIR` pointing to temp folders
  (`docs/DEVELOPMENT.md`, "Using roundhouse while developing it"). Test the mod with `claude --plugin-dir mod`.
- Tests must not need Claude Code, a terminal or the network: the list is tested by feeding `tea.Msg` values and reading `View()`, the update path
  against an `httptest` server, starters by building the command only. A real terminal and a real `claude` are needed only for `docs/MANUAL_TESTS.md`.

## Releases

Tag `vX.Y.Z` on a green `main`; the release workflow builds archives, `.deb`/`.rpm` and a cosign-signed `checksums.txt` (GoReleaser). Then update the table
"What is verified where" in the README. The first tag a user runs with the update feature is the first one they can be told about: a build without the
update check (0.2.0 and older) never reports newer versions, so users on it update once by hand.

## Gotchas

- `go test -race` needs cgo; on a plain Windows setup rely on CI.
- Windows replaces a running `.exe` by renaming it to `.old` first; `update.CleanOld` removes it at a later start.
- Dependabot opens pull requests for Go modules and Actions. `x/ansi` and `cellbuf` must move together with `bubbles`; the `cosign-installer` v4 bump has not
  been tried on a real release yet (CI skips signing), so try it on a release with nothing else changing.
- `go-licenses` cannot detect `mattn/go-localereader` v0.0.1 (no license file; upstream is MIT); it is excluded in CI and listed by hand.
