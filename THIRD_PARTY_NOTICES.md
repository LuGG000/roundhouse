# Third-party notices

roundhouse is licensed under the Apache License 2.0 (see `LICENSE`). The binary includes these
Go modules, all under permissive licenses. Regenerate this list with
`go run github.com/google/go-licenses/v2@v2.0.1 report ./cmd/roundhouse --ignore github.com/mattn/go-localereader`
(run once with `GOOS=windows` and once with `GOOS=linux`) when dependencies change; CI checks that every
dependency uses MIT, BSD-2-Clause, BSD-3-Clause, Apache-2.0 or ISC.

| Module | License |
| --- | --- |
| [github.com/atotto/clipboard](https://github.com/atotto/clipboard/blob/v0.1.4/LICENSE) | BSD-3-Clause |
| [github.com/aymanbagabas/go-osc52/v2](https://github.com/aymanbagabas/go-osc52/blob/v2.0.1/LICENSE) | MIT |
| [github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles/blob/v0.21.0/LICENSE) | MIT |
| [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea/blob/v1.3.10/LICENSE) | MIT |
| [github.com/charmbracelet/colorprofile](https://github.com/charmbracelet/colorprofile/blob/f60798e515dc/LICENSE) | MIT |
| [github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss/blob/v1.1.0/LICENSE) | MIT |
| [github.com/charmbracelet/x/ansi](https://github.com/charmbracelet/x/blob/ansi/v0.10.1/ansi/LICENSE) | MIT |
| [github.com/charmbracelet/x/cellbuf](https://github.com/charmbracelet/x/blob/2c3ea96c31dd/cellbuf/LICENSE) | MIT |
| [github.com/charmbracelet/x/term](https://github.com/charmbracelet/x/blob/term/v0.2.1/term/LICENSE) | MIT |
| [github.com/erikgeiser/coninput](https://github.com/erikgeiser/coninput/blob/1c3628e74d0f/LICENSE) | MIT |
| [github.com/inconshreveable/mousetrap](https://github.com/inconshreveable/mousetrap/blob/v1.1.0/LICENSE) | Apache-2.0 |
| [github.com/lucasb-eyer/go-colorful](https://github.com/lucasb-eyer/go-colorful/blob/v1.2.0/LICENSE) | MIT |
| [github.com/mattn/go-isatty](https://github.com/mattn/go-isatty/blob/v0.0.20/LICENSE) | MIT |
| [github.com/mattn/go-localereader](https://github.com/mattn/go-localereader/blob/master/LICENSE) | MIT (1) |
| [github.com/mattn/go-runewidth](https://github.com/mattn/go-runewidth/blob/v0.0.16/LICENSE) | MIT |
| [github.com/muesli/ansi](https://github.com/muesli/ansi/blob/276c6243b2f6/LICENSE) | MIT |
| [github.com/muesli/cancelreader](https://github.com/muesli/cancelreader/blob/v0.2.2/LICENSE) | MIT |
| [github.com/muesli/termenv](https://github.com/muesli/termenv/blob/v0.16.0/LICENSE) | MIT |
| [github.com/pelletier/go-toml/v2](https://github.com/pelletier/go-toml/blob/v2.4.3/LICENSE) | MIT |
| [github.com/rivo/uniseg](https://github.com/rivo/uniseg/blob/v0.4.7/LICENSE.txt) | MIT |
| [github.com/spf13/cobra](https://github.com/spf13/cobra/blob/v1.10.2/LICENSE.txt) | Apache-2.0 |
| [github.com/spf13/pflag](https://github.com/spf13/pflag/blob/v1.0.9/LICENSE) | BSD-3-Clause |
| [github.com/xo/terminfo](https://github.com/xo/terminfo/blob/abceb7e1c41e/LICENSE) | MIT |
| [golang.org/x/sys](https://cs.opensource.google/go/x/sys/+/v0.44.0:LICENSE) | BSD-3-Clause |
| [golang.org/x/text](https://cs.opensource.google/go/x/text/+/v0.41.0:LICENSE) | BSD-3-Clause |

(1) The pinned version v0.0.1 of go-localereader has no license file, so `go-licenses` cannot detect it.
The project's current `LICENSE` is MIT, and it is excluded from the automated check for that reason.
