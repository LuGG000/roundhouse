# roundhouse-mod

Optional mod for Claude Code. It talks to the `roundhouse` command only (`list --json --folder`, `open --terminal`), so roundhouse works without it.

One command, `/roundhouse`, opens a pane with the chats of the folder the session runs in, newest first:

- **Switch here** continues in this session with that chat, as if you had typed `/resume <id>`.
- **New window** opens the chat in a new terminal window, in its folder, with your configured roundhouse starter (so it also works through Switchyard).
- **New chat** starts a fresh conversation in this session (`/clear`).
- **Refresh** reads the folder again. The chat you are in is marked "this chat". The pane also follows a switch, a new chat and every finished turn by itself.
- **Style** opens a page like switchyard-mod's: palettes (default, dark, light) and four colors (background, text, accent, borders) with a picker of common colors and a field for any other one (`#00ff88`, a color name, `default`, or `<slot> <color>`). It only changes this pane; the colors are kept in the mod's own store, so they survive restarts. `/roundhouse style` opens the page directly.
- **Update notice:** when `roundhouse` knows a newer release (from its daily check, `update_check` in its config), the pane says so, with the version you have, and a toast mentions it once per session. **Update…** opens a new terminal window that runs `roundhouse update`, where you answer the question; the mod never installs anything itself, and nothing changes before you say yes there. The mod itself is updated by Claude Code: `claude plugin update roundhouse-mod@roundhouse`.

A mod cannot change the folder of a running session, so chats of other folders are not listed here. Use `roundhouse` in a terminal for all folders.

If `roundhouse` is missing from `PATH` or reports a `list` schema the mod does not read, the mod stays quiet and the pane says why.

## Install

You need roundhouse 0.2.0 or newer on your `PATH` (`roundhouse list --folder` and `roundhouse open --terminal` are new there) and Claude Code 2.1.287 or newer. It is made for the terminal. In the Code tab of the desktop app the pane opens, but the session there has no `/resume`, so **Switch here** opens the chat in a new window instead; buttons may also need a second click there (not understood yet).

```
roundhouse mod install      # or, inside Claude Code: /plugin install roundhouse-mod --marketplace LuGG000/roundhouse
roundhouse mod uninstall    # switch it off again
```

The mod is optional, and installing it is the switch: an installed mod is on in every Claude Code session you start from then on, in every Switchyard profile too (the plugins folder is shared), so a chat you open with **New window** has `/roundhouse` as well. Remove it with `roundhouse mod uninstall`, or turn it off for a while with `/plugin` inside Claude Code. `roundhouse mod install --source <path>` installs from a local checkout; `--scope project` limits it to one project.

Mods are not sandboxed and run with your user rights; this one only starts the `roundhouse` command and the slash commands named above.

To try it from a clone in one session only (a chat opened with **New window** will not have it then):

```
claude --plugin-dir mod
```

## Notes

- The pane appears on the side or above the prompt, as Claude Code decides for its width. Opened with `/roundhouse` it shows at any width.
- Chats are read through `roundhouse`, so there is no 4 MiB file limit and the list is the same as in the terminal.
- It does not use `holdToasts`, a status line, or `--settings`, so it can sit beside other mods such as switchyard-mod.
- With switchyard-mod loaded as well, Claude Code shows both panes as tabs with their titles (`roundhouse` and switchyard's) and you switch between them there; the mods do not know about each other.

## Development

```
claude plugin validate mod
claude plugin test mod
```

The types (`mod/.claude-plugin/types`, not committed) are written by Claude Code when it loads the mod.
