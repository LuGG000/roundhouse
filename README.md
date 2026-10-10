# roundhouse

A hub for [Claude Code](https://code.claude.com): start a new chat or return to an old one, in any folder, from one list. Claude Code's own `/resume` only shows the chats of the current folder.

> Early development. Not affiliated with or endorsed by Anthropic.

roundhouse reads the session transcripts Claude Code keeps under `~/.claude/projects`, shows them grouped by folder, and starts Claude Code in the right folder when you pick one. It never writes to your chats.

## Usage

```
roundhouse                 open the interactive list
roundhouse list [--json]   print the chats, newest first
roundhouse open <id>       resume a chat in its folder (a unique id prefix is enough)
roundhouse new [folder]    start a new chat in a folder (default: the current one)
```

In the list, use the mouse or the keyboard:

| Input | Action |
|---|---|
| click, `enter` | open the chat (the first click selects, a click on the selected chat opens it) |
| `n` | new chat: pick a folder |
| `g` | group by folder or show one flat list |
| `/` | filter by title, folder or last prompt |
| `r` | reload |
| `q`, `esc` | quit |

`list --json` prints `{"schema": 1, "sessions": [...]}` for other tools.

## Starters

A starter is the command that launches Claude Code. Choose one with `starter` in the config or `--starter` on the command line.

| Starter | Resume | New chat |
|---|---|---|
| `claude` (default) | `claude --resume <id>` | `claude` |
| `switchyard` | `switchyard run -- --resume <id>` | `switchyard run` |

Add your own, or override a built-in one, in `config.toml`. The arguments may use `{id}`, `{path}` (the transcript file) and `{cwd}`; the command always runs in the chat's folder.

```toml
starter = "wrapped"

[[starters]]
name = "wrapped"
resume = ["my-wrapper", "--resume", "{id}"]
new = ["my-wrapper"]
```

Config file: `<user config dir>/roundhouse/config.toml` (`%AppData%\roundhouse` on Windows, `~/.config/roundhouse` on Linux). Override the folder with `ROUNDHOUSE_CONFIG_DIR`.

## Switchyard

roundhouse works with [Switchyard](https://github.com/LuGG000/switchyard). With the `switchyard` starter every chat runs under `switchyard run`, so limit failover keeps working. roundhouse does not touch Switchyard's profiles, settings or credentials.

Switchyard links `projects/` of every profile to the default one, so all chats of all profiles show up in the list.

## Notes

- The folder of a chat is the first `cwd` in its transcript. Resuming a chat from another folder adds later lines with that other folder, but roundhouse always starts in the original one.
- Claude Code deletes transcripts older than `cleanupPeriodDays` (30 by default). Chats removed that way are gone from the list.
- Chats are read from `CLAUDE_CONFIG_DIR/projects`, or `~/.claude/projects` when it is not set.
- The transcript format is not documented by Anthropic. Unknown lines are ignored, but a Claude Code update can still change what roundhouse finds.

## Build

```
go build ./cmd/roundhouse
go test ./...
```

Requires Go 1.26 or newer. Linux and Windows are tested, macOS builds.

## License

[Apache-2.0](LICENSE)
