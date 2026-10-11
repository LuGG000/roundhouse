# Architecture

roundhouse is a single Go binary. Its only network request is the optional daily look at the public release list (see "Updates"), and it never starts anything on its own: a
chat is started only when you pick it.

## Packages

| Package | Job |
|---|---|
| `cmd/roundhouse` | cobra commands (`list`, `open`, `new`, and the list itself), exit-code handling |
| `internal/transcript` | reads one transcript file and returns id, first `cwd`, title and last prompt |
| `internal/index` | scans `projects/*/*.jsonl` with a worker pool, sorts newest first, keeps the index cache |
| `internal/favorites` | starred chats and folders in `favorites.json` |
| `internal/starter` | builds the command that resumes a chat or starts a new one |
| `internal/config` | loads `config.toml` and resolves the active starter |
| `internal/terminal` | builds the command that opens a program in a new terminal window (`open --terminal`) |
| `internal/shortcut` | writes and removes the application launcher entry (`shortcut install`): `.lnk`, `.desktop` file or `.app` bundle |
| `internal/update` | finds the latest release (cached for a day) and installs a chosen version after checking its checksum |
| `internal/paths` | locates the Claude, config and data folders |
| `internal/tui` | the bubbletea model: items, layout, keys, mouse |

## Data flow

```
~/.claude/projects/*/*.jsonl --> transcript.Read --> index.Scan (+ cache) --> tui.Model
                                                                                |
                    favorites.json <------------ f / F ------------------------+
                                                                                |
                         config.toml --> starter --> exec.Cmd (Dir = chat folder) +--> claude
```

- `transcript.Read` scans a file line by line. Only three kinds of lines matter: `ai-title` (generated title),
  `last-prompt` and the first line that carries a top-level `cwd`. Everything else is skipped without being
  decoded. Lines of any size are fine; a line over 256 MiB ends the scan with what was found so far.
- `subagents/` and other nested folders are not visited; only `projects/<project>/<id>.jsonl` are chats.
- A chat without a folder or a title (for example one that was opened and left at once) is not listed.

## Files roundhouse writes

| File | Content |
|---|---|
| `<data dir>/favorites.json` | ids of starred chats and paths of starred folders |
| `<data dir>/index-cache.json` | per transcript: size, modification time, id, folder, title, last prompt |

Both are written atomically (temp file, then rename). The cache is rebuilt when it is missing, unreadable
or from another version; it only costs speed. roundhouse never writes inside the Claude folder.

## The list

`tui.Model` holds the sessions and a flat slice of `item`s that is rebuilt after every change of data, filter
or view. An item is a section header, a chat, a section footer or (on the folder screen) a folder. Heights
are computed, not stored: headers take one line, footers two, chats two, and the selected chat four lines
(the card). Scrolling, mouse hit testing and rendering all use the same height function.

Starting a chat uses `tea.ExecProcess`, which hands the terminal to the child and takes it back afterwards.
roundhouse never kills the child, so the terminal state is restored by bubbletea. The exit code of the child
is not an error; only a failure to start it is shown.

## Starters

A starter has an argument list for resuming and one for a new chat, with the placeholders `{id}`, `{path}`
and `{cwd}`. The command always runs with the chat's original folder as working directory. Starting fails
with a message when that folder no longer exists.

## The mod

`mod/` is a Claude Code mod (a plugin with in-process hooks). It never reads transcripts and never writes files; it
only runs the `roundhouse` command through `$.process.run`:

| What the pane does | Call |
|---|---|
| list the chats of the session's folder | `roundhouse list --json --folder <cwd>` (schema 1) |
| open a chat in a new window | `roundhouse open <id> --terminal` |
| switch this session to a chat | the slash command `/resume <id>` through `$.command.run` |
| new chat | the slash command `/clear` through `$.command.run` |

A mod cannot restart Claude Code or change the folder of a running session, which is why the pane shows the chats of
one folder only. `open --terminal` starts a new terminal window running `roundhouse open <id> --starter <name>` in the
chat's folder (Windows Terminal, else `cmd start`; Terminal.app; or the first Linux terminal emulator found), so the
starter, including Switchyard, is the same as in the list.

The pane's colors are the mod's own business: four slots (background, text, accent, borders) kept as one JSON value in
the mod's `$.store`, which lives in the Claude plugin store and survives restarts. roundhouse itself has no theme setting.
The pane refreshes after its own actions and, while it is open, after every finished turn and at session start, so
the "this chat" marker follows a `/resume` without a button press.

The pane reads the colors from the store every time it draws. The session state (`$.state`) can start over, for
example after a `/resume`, while the store stays, so the store is the source of truth and the state value only
tells the pane to draw again after a change.

## Updates

`internal/update` has two halves that are only joined by the `update` command, after the user said yes:

- `Checker` reads `releases/latest` of the repository and keeps the answer in `update-check.json` in the data folder for a day; a
  failed lookup is remembered as well, so an offline machine does not retry on every command. `Cached` reads the file without the
  network. A build whose version is not a release number (`dev`) is never considered outdated.
- `Installer.Install(version, target)` fetches `releases/tags/v<version>`, takes the archive for the running system and
  `checksums.txt` from it, refuses any file that is not a release download of this repository, compares the SHA-256, extracts
  the binary and replaces `target`. On Windows the running file is renamed to `.old` first (deleted at a later start), elsewhere
  the new file is renamed over it. It installs exactly the version the user was shown, not "the latest by then".

The notices read only the cache, plus one refresh per day within three seconds: the list asks in the background, `list --json`
(and so the mod) asks when the cache is older than a day. `update_check = false` turns all of that off; `update` itself is an
explicit request and works either way. The mod only shows the notice and starts `roundhouse update --terminal`, which opens a
window where the question is asked; it never installs anything itself.
