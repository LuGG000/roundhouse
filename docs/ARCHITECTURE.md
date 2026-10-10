# Architecture

roundhouse is a single Go binary. It never talks to the network and never starts anything on its own: a
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
