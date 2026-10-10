# roundhouse

[![CI](https://github.com/LuGG000/roundhouse/actions/workflows/ci.yml/badge.svg)](https://github.com/LuGG000/roundhouse/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/LuGG000/roundhouse)](https://github.com/LuGG000/roundhouse/releases/latest)
[![License](https://img.shields.io/github/license/LuGG000/roundhouse)](LICENSE)

Open-source hub for [Claude Code](https://code.claude.com): one list of all your chats, grouped by
folder. Start a new chat or return to an old one in any folder, with the mouse or the keyboard.
Claude Code's own `/resume` only shows the chats of the current folder.

**Status: early development.** The list, favorites and starting chats have been used with the real
`claude` on Windows 11 (Windows Terminal). Linux and macOS are built and tested in CI only; see
[Scope](#scope) for what is verified where.

## How it works

- Claude Code keeps every chat as a transcript file under `~/.claude/projects/<project>/<id>.jsonl`.
  roundhouse reads those files (never writes to them) and takes the title, the folder and the
  time of each chat from them.
- Picking a chat starts Claude Code in the chat's original folder with `claude --resume <id>`.
  Which command is used is set by a [starter](#starters), so wrappers such as
  [Switchyard](#switchyard) work too.
- Favorites (chats and folders) and a small index cache are kept in roundhouse's own data folder.
  Nothing is sent anywhere.

## Quick start

Install roundhouse (next section), then:

```
roundhouse                # open the list
roundhouse new            # start a new chat in the current folder
roundhouse list --json    # the chats, for scripts
```

## Install

You need Claude Code (`claude` on your `PATH`, tested with 2.1.296) on Windows, Linux or macOS (see
[Scope](#scope)); Go 1.26 or newer only to build from source. A step-by-step guide is in
[docs/SETUP.md](docs/SETUP.md).

From source (Go 1.26 or newer):

```
go install github.com/LuGG000/roundhouse/cmd/roundhouse@latest
```

Prebuilt archives for Linux, macOS and Windows (amd64, arm64) and `.deb` and `.rpm` packages are
built by the release workflow and will be on the
[releases page](https://github.com/LuGG000/roundhouse/releases) from the first release on.

### Verify a download

Each release has `checksums.txt` with the SHA-256 of every file and a keyless
[cosign](https://github.com/sigstore/cosign) signature of it, made by the release workflow.

```
cosign verify-blob checksums.txt --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp '^https://github.com/LuGG000/roundhouse/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
sha256sum --check --ignore-missing checksums.txt
```

## Usage

```
roundhouse                        open the interactive list
roundhouse list [--json]          print the chats, newest first
                [--folder <dir>]  only the chats started in that folder
roundhouse open <id>              resume a chat in its folder (a unique id prefix is enough)
                [--terminal]      in a new terminal window, and return at once
roundhouse new [folder]           start a new chat in a folder (default: the current one)
roundhouse --version
```

All commands accept `--starter <name>` to override the configured starter.

In the list, use the mouse or the keyboard:

| Input | Action |
|---|---|
| click, `enter` | open the chat (the first click selects, a click on the selected chat opens it) |
| `n` | new chat: pick a folder |
| `f` | star or unstar the chat; favorite chats get their own section at the top |
| `F` | star or unstar the folder of the selected chat; favorite folders are listed first |
| `g` | group by folder or show one flat list |
| `/` | filter by title, folder or last prompt |
| `r` | reload |
| `q`, `esc` | quit |

On the folder screen (`n`), `f` stars the selected folder and `enter` starts a new chat in it.

`list --json` prints `{"schema": 1, "sessions": [...]}`; the schema number changes when fields are
removed or change meaning.

### Exit codes

`open` and `new` pass on the exit code of the started program. `1` means roundhouse itself failed
(unknown id, missing folder, unreadable config).

## Configuration

Config file: `<user config dir>/roundhouse/config.toml` (`%AppData%\roundhouse` on Windows,
`~/.config/roundhouse` on Linux, `~/Library/Application Support/roundhouse` on macOS). Override the
folder with `ROUNDHOUSE_CONFIG_DIR`. Without a file the defaults apply.

| Key | Meaning | Default |
|---|---|---|
| `starter` | name of the starter that launches Claude Code | `claude` |
| `[[starters]]` | custom starters, see below | none |

Data folder (favorites, index cache): `$XDG_DATA_HOME/roundhouse`, else `%LocalAppData%\roundhouse`,
else `~/.local/share/roundhouse`. Override with `ROUNDHOUSE_DATA_DIR`. Chats are read from
`$CLAUDE_CONFIG_DIR/projects`, or `~/.claude/projects` when it is not set.

### Starters

A starter is the command that launches Claude Code. Choose one with `starter` in the config or
`--starter` on the command line.

| Starter | Resume | New chat |
|---|---|---|
| `claude` (default) | `claude --resume <id>` | `claude` |
| `switchyard` | `switchyard run -- --resume <id>` | `switchyard run` |

Add your own, or override a built-in one, in `config.toml`. The arguments may use `{id}`, `{path}`
(the transcript file) and `{cwd}`; the command always runs in the chat's folder.

```toml
starter = "wrapped"

[[starters]]
name = "wrapped"
resume = ["my-wrapper", "--resume", "{id}"]
new = ["my-wrapper"]
```

## Switchyard

roundhouse works with [Switchyard](https://github.com/LuGG000/switchyard). With the `switchyard`
starter every chat runs under `switchyard run`, so limit failover keeps working. roundhouse does not
touch Switchyard's profiles, settings or credentials and does not read its files.

Switchyard links `projects/` of every profile to the default one, so all chats of all profiles show
up in the list.

## Optional mod

[`mod/`](mod/README.md) is a Claude Code mod. `/roundhouse` opens a pane with the chats of the folder the session runs in: switch to one with a click, open it in a new terminal window, or start a new chat. It talks to the `roundhouse` command only, so roundhouse works without it.

```
/plugin install roundhouse-mod --marketplace LuGG000/roundhouse
```

It needs roundhouse 0.2.0 or newer on the `PATH`. A mod cannot change the folder of a running session, so the pane shows one folder; the terminal list shows all of them.

## Documentation

- [docs/SETUP.md](docs/SETUP.md): install and first run
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): packages, data flow, what is written where
- [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md): build, test, workflow
- [docs/SPIKE.md](docs/SPIKE.md): what was verified against the real Claude Code
- [docs/MANUAL_TESTS.md](docs/MANUAL_TESTS.md): checks that need a real terminal
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
- [mod/README.md](mod/README.md): the Claude Code mod

## Scope

roundhouse reads Claude Code's undocumented transcript files. Unknown lines are ignored, but a
Claude Code update can still change what roundhouse finds.

### What is verified where

| | Windows 11 | Linux | macOS |
|---|---|---|---|
| Unit tests (CI) | yes | yes (with `-race`) | yes |
| Interactive list in a real terminal | yes (Windows Terminal) | not yet | not yet |
| Starting `claude` from the list | yes | not yet | not yet |
| Starting through Switchyard | yes | not yet | not yet |
| Mod pane in Claude Code | not yet | not yet | not yet |
| Opening a chat in a new terminal window | not yet | not yet | not yet |

WSL is not verified. Other terminals than Windows Terminal are not verified; mouse support needs a
terminal that reports mouse events.

### Notes

- The folder of a chat is the first `cwd` in its transcript. Resuming a chat from another folder
  adds later lines with that other folder, but roundhouse always starts in the original one.
- Claude Code deletes transcripts older than `cleanupPeriodDays` (30 by default). Chats removed
  that way are gone from the list.

## Non-goals

- Editing, renaming or deleting chats.
- Managing logins or switching accounts (that is Switchyard's job).
- Full-text search in the chat content.

## Uninstall

Remove the binary and, if you want, the config and data folders named under
[Configuration](#configuration). roundhouse changes nothing else on your system.

## License

[Apache-2.0](LICENSE). Third-party licenses: [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
Not affiliated with or endorsed by Anthropic.
