# Troubleshooting

## The list is empty ("no chats found")

- roundhouse reads `$CLAUDE_CONFIG_DIR/projects`, or `~/.claude/projects`. Check that the folder exists and holds
  `<project>/<id>.jsonl` files. If you start Claude Code with another `CLAUDE_CONFIG_DIR`, set the same one when you
  start roundhouse.
- Claude Code deletes chats older than `cleanupPeriodDays` (30 days by default). Raise it in `~/.claude/settings.json`
  if you want to keep chats longer; roundhouse does not change this setting.
- Chats that were opened but never got a prompt have no title and are not listed.

## "folder ... does not exist"

The chat belongs to a folder that was moved or deleted. Restore the folder or move it back; roundhouse starts
the chat in its original folder only.

## "could not start: exec: "claude": executable file not found"

`claude` (or the program of your starter) is not on the `PATH` of the terminal you started roundhouse from. Check
with `claude --version`, or change the `starter` in the config.

## "unknown starter"

The `starter` in `config.toml` or `--starter` names nothing: use `claude`, `switchyard` or the `name` of a
`[[starters]]` entry.

## The mouse does nothing, or the box drawing characters look broken

Use a terminal that reports mouse events and has a font with box drawing characters (Windows Terminal, iTerm2,
most Linux terminals). Inside tmux, enable `set -g mouse on`. The keyboard works everywhere.

## The list looks wrong after a Claude Code update

The transcript format is not documented. Run `roundhouse list --json` and compare with the files; if titles or
folders are missing, open an issue with the `claude --version` and a redacted example line.

## A chat shows an old title or folder

roundhouse keeps an index cache and re-reads a transcript when its size or modification time changes. To force a
full rescan, delete `index-cache.json` in the data folder (see "Configuration" in the README).

## "could not save favorites"

The data folder is not writable. Check its permissions, or set `ROUNDHOUSE_DATA_DIR` to a writable folder.

## The mod says "roundhouse is not available"

The `roundhouse` command is not on the `PATH` of the terminal Claude Code was started from, or it is older than 0.2.0
(`list --folder` and `open --terminal` are new there). Check with `roundhouse --version`.

## The mod's "Switch here" does nothing

It runs `/resume <id>` as if you had typed it, once the session is idle. If a turn is still running, it waits for the end of
the turn. Chats of other folders are not listed in the pane, because a session cannot change its folder; use
**New window** or the terminal list.

## "New window" opens nothing

`roundhouse open <id> --terminal` needs Windows Terminal or `cmd` on Windows, Terminal.app on macOS, or one of
`x-terminal-emulator`, `gnome-terminal`, `konsole`, `xfce4-terminal` and `xterm` on Linux. Run the command in a shell to see
the error.

## `/roundhouse` is missing in a chat opened with "New window"

The mod was loaded for one session only (`claude --plugin-dir mod`), and the new window starts a plain `claude`. Install the mod
with `roundhouse mod install`: an installed mod is on in every new session. Check with `claude plugin list`, which should show
`roundhouse-mod@roundhouse` as enabled.
