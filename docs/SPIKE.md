# Spike: what was verified against Claude Code

Facts about Claude Code that roundhouse relies on. Everything below was run against the real `claude`
2.1.296 on Windows 11 on 2026-10-10, with `--model haiku` and short prompts in empty temp folders.
Claude Code does not document the transcript format, so re-check after major updates.

## Summary

| # | Question | Result |
|---|---|---|
| a | Do transcript lines carry the working folder? | verified: `user` and `attachment` lines have a top-level `cwd`; meta lines (`mode`, `file-history-snapshot`, `last-prompt`, ...) do not |
| b | Where do titles live? | verified: `ai-title` lines (`aiTitle`) appear several times in a chat, the last one is current; `last-prompt` lines (`lastPrompt`) hold the latest prompt |
| c | Does `claude -p --resume <id>` work from another folder? | verified: yes |
| d | Does `claude -p --resume <path to .jsonl>` work from another folder? | verified: yes |
| e | What happens to the transcript when a chat is resumed from another folder? | verified: it stays one file in the original project folder, and later lines carry the new folder |
| f | Are transcripts bigger than 4 MiB common? | verified: one of 31 chats had 22 MB |
| g | Does `/rename` store a title in the transcript? | not verified |
| h | Does an interactive `claude --resume <id>` also work from another folder? | not verified; roundhouse avoids the question by always starting in the original folder |

## Notes

### (a) and (e): the folder of a chat

A chat created in folder A and resumed from folder B ended up as one file under the project folder of A, with
19 lines saying `cwd` A and 7 lines saying `cwd` B. So the last `cwd` is not reliable; roundhouse uses the
first one, which matches the project folder the file lives in.

The name of the project folder (`C--Users-name-...`) is the path with separators replaced, which cannot be
turned back into a path. roundhouse never tries to.

### (b): titles

Of the line types seen in real transcripts, only `ai-title` and `last-prompt` are used. A chat that was never
long enough to get a generated title shows its last prompt instead.

### (c) and (d): resuming from another folder

Both forms worked in non-interactive mode from an unrelated folder. roundhouse still starts the process in the
chat's folder so that Claude Code's file access, `CLAUDE.md` and permissions match the project.

### (f): big transcripts

roundhouse streams each file line by line, so the size of a transcript is no problem.
The index cache skips re-reading unchanged files.

## Method

```
mkdir folderA folderB
cd folderA && claude -p "Answer only OK" --model haiku --output-format json     # prints session_id
cd ../folderB && claude -p "Say OK" --model haiku --resume <session_id>         # (c)
cd ../folderB && claude -p "Say OK" --model haiku --resume <path to the .jsonl> # (d)
grep -o '"cwd":"[^"]*"' <the .jsonl> | sort | uniq -c                            # (e)
```

The test chats were deleted afterwards (`~/.claude/projects/*folderA`).
