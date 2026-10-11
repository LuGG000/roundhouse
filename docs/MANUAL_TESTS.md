# Manual tests

Checks that need a real terminal and a real `claude`. Use `--model haiku`, short prompts and a throwaway
folder for the chats you create. Run them before a release, once per platform you claim in the README.

## The list

1. `roundhouse` shows your chats grouped by folder; the selected chat is a card with an accent border.
2. Arrow keys, `j`/`k`, `pgup`/`pgdn`, `home`/`end` move one chat at a time and skip headers and footers.
3. The mouse wheel moves exactly one chat per notch.
4. A click selects a chat; a click on the selected chat opens it.
5. `/` filters while typing; `esc` clears the filter; `enter` keeps it.
6. `g` switches between folder sections and one flat list.
7. Resize the window to about 40 and 120 columns: no line wraps or overflows, the title line is cut cleanly.
8. Light and dark terminal themes both read well; with `NO_COLOR=1` the list is still usable.

## Starting chats

1. Open a chat of another folder: `claude` starts in that folder (`pwd` / `cd` in a shell command shows it) and
   shows the old conversation.
2. Quit `claude` with `/exit` and with Ctrl+C: the list comes back, keyboard and mouse still work, no stray
   characters on screen.
3. `n`, pick a folder, `enter`: a new chat starts there.
4. Rename or delete the folder of a chat, then open the chat: the list shows "folder ... does not exist" and stays usable.
5. `roundhouse open <id prefix>` and `roundhouse new <folder>` work outside the list and pass on the exit code.

## Favorites

1. `f` on a chat: it appears in the "Favorites" section at the top and keeps its star after restarting roundhouse.
2. `F` on a chat: its folder section moves to the top with a star; `F` again removes it.
3. On the folder screen (`n`), `f` stars the selected folder and moves it to the top.
4. Make the data folder read-only: `f` shows "could not save favorites" and the star is not set.

## With Switchyard

1. `roundhouse --starter switchyard`: opening a chat starts `switchyard run -- --resume <id>` in the chat's folder.
2. The failover still works for a chat started this way (needs a profile at its limit or a simulated limit; see
   Switchyard's `docs/MANUAL_TESTS.md`).

## The mod

Start Claude Code with `claude --plugin-dir mod` in a folder that has chats, with the new roundhouse on the `PATH`. Then, with a throwaway Claude config (`CLAUDE_CONFIG_DIR` set to an empty folder): `roundhouse mod install --source .` shows the mod as enabled in `claude plugin list`, a new `claude` session has `/roundhouse`, and `roundhouse mod uninstall` removes it again.

1. `/roundhouse` opens the pane at 100 and at 160 columns, and the chats of this folder are listed, the current one marked.
2. **Switch here** on another chat continues this session with that chat (`/resume`); the transcript shows it.
3. **New window** opens a terminal window in the chat's folder that shows the old conversation; closing it leaves the session alone.
4. **New chat** starts a fresh conversation (`/clear`).
5. After **Switch here** and **New chat** the pane updates without pressing Refresh; **Refresh** also picks up a chat created in another terminal.
   **Style** changes the palette and single colors, and they are still set after restarting Claude Code.
6. Rename the `roundhouse` binary: the pane says it is not available and the session works as before.
7. Together with switchyard-mod: both panes open as tabs you can switch between, neither hides the other's toasts, and a limit still opens its own card.
8. The desktop app's Code tab shows the same pane.

## Updates

Use a build with an older release number, for example `go build -ldflags "-X main.version=0.1.0" -o roundhouse-dev ./cmd/roundhouse`, with a throwaway `ROUNDHOUSE_DATA_DIR`.

1. `roundhouse-dev update --check` names the latest release and says you have the older one; `--check --json` prints the same as JSON.
2. `roundhouse-dev` shows a line at the bottom of the list that a newer release is available, and nothing else changes.
3. `roundhouse-dev update` shows the version, the notes link and the path it would replace, then asks. Answer `n`, press Enter or close the window: nothing changes. Piping an answer (`echo y | roundhouse-dev update`) is refused.
4. Answer `y`: the binary is replaced, `--version` shows the new number, and on Windows `roundhouse.exe.old` is gone at the next start.
5. With `update_check = false` neither the list nor `list --json` mention an update, but `update --check` still works.
6. In the mod, the pane shows the notice and **Update…**; it opens a window that asks the same question, and the pane itself changes nothing.

## App launcher entry

Use a throwaway profile so your own entry stays: on Windows set `APPDATA` to an empty folder, on Linux `XDG_DATA_HOME`, on macOS
use a throwaway user.

1. `roundhouse-dev shortcut install` prints the path it created and nothing else changes.
2. Find roundhouse by name (Start menu search, the app menu of the desktop, Spotlight) and start it: a terminal window with the list opens.
3. `roundhouse-dev shortcut remove` deletes the entry; running it again says there is nothing to remove.

## Platforms

Repeat "The list" and "Starting chats" on every platform and terminal named as verified in the README
("What is verified where"), and update that table.
