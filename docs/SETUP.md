# Setup

## Requirements

- Claude Code installed, with `claude` on your `PATH` (tested with 2.1.296). Check with `claude --version`.
- Windows, Linux or macOS. On Windows, use Windows Terminal; it reports mouse events and shows the box
  drawing characters correctly.
- Go 1.26 or newer, only to build from source.

## Install

From source:

```
go install github.com/LuGG000/roundhouse/cmd/roundhouse@latest
```

This puts `roundhouse` into `$(go env GOPATH)/bin` (`%USERPROFILE%\go\bin` on Windows). Add that folder
to your `PATH` if it is not there yet.

From a release archive (once a release exists): unpack it and put `roundhouse` (`roundhouse.exe` on
Windows) in a folder on your `PATH`. On Debian and Fedora derivatives the `.deb` and `.rpm` packages do
the same. See "Verify a download" in the README to check the signature.

## First run

```
roundhouse
```

You should see your chats grouped by folder. If the list is empty, see
[TROUBLESHOOTING.md](TROUBLESHOOTING.md).

Pick a chat with the mouse or with the arrow keys and `enter`. Claude Code starts in the folder of that
chat. When you leave Claude Code, you are back in the list.

## Using it with Switchyard

If you use [Switchyard](https://github.com/LuGG000/switchyard) for several logins, start chats through it so
that the limit failover keeps working:

```
roundhouse --starter switchyard
```

or make it the default in `config.toml` (see "Configuration" in the README):

```toml
starter = "switchyard"
```

## Start it from anywhere

The list always starts chats in their own folder, so you can run `roundhouse` from any folder. For a quick
way to open it, bind `roundhouse` to a shortcut or a Windows Terminal profile, or add an alias in your shell.

## The optional mod

```
roundhouse mod install
```

adds the roundhouse marketplace to Claude Code and installs `roundhouse-mod` from it. From then on `/roundhouse` is available in every new
Claude Code session, also in chats you open from the list or from the mod's **New window**. Leave it out, or run
`roundhouse mod uninstall`, if you do not want it; roundhouse works the same without it. See [mod/README.md](../mod/README.md).

## Updating

roundhouse tells you when a newer release exists (see "Updates" in the README). `roundhouse update` shows it and installs it after you
answer yes; or download the archive yourself as in "Install". The mod is updated by Claude Code:
`claude plugin update roundhouse-mod@roundhouse`.
