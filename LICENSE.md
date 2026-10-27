# License

## Kalimate

MIT License

Copyright (c) 2026 niespodd

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Third-party dependencies

Kalimate is distributed as a single binary that statically links the Go
modules below. Every one is under a permissive license (MIT, BSD, ISC,
Apache-2.0, or public domain) — all compatible with Kalimate's MIT license.
There is **no copyleft** (GPL / LGPL / MPL) code in the shipped binary.

Each dependency retains its own copyright and license; the full texts live in
the Go module cache (`go env GOMODCACHE`) and in each project's repository.

| Module | License |
| --- | --- |
| fyne.io/fyne/v2 | BSD-3-Clause |
| fyne.io/systray | Apache-2.0 |
| github.com/BurntSushi/toml | MIT |
| github.com/anthonynsimon/bild | MIT |
| github.com/clipperhouse/uax29/v2 | MIT |
| github.com/fsnotify/fsnotify | BSD-3-Clause |
| github.com/fyne-io/image | BSD-3-Clause |
| github.com/fyne-io/oksvg | BSD-3-Clause |
| github.com/gen2brain/malgo | Unlicense (public domain) |
| github.com/go-gl/gl | MIT |
| github.com/go-gl/glfw/v3.4/glfw | BSD-3-Clause |
| github.com/go-text/render | BSD-3-Clause |
| github.com/go-text/typesetting | BSD-3-Clause |
| github.com/godbus/dbus/v5 | BSD-2-Clause |
| github.com/jeandeaual/go-locale | MIT |
| github.com/jsummers/gobmp | MIT |
| github.com/mattn/go-runewidth | MIT |
| github.com/nfnt/resize | ISC |
| github.com/nicksnyder/go-i18n/v2 | MIT |
| github.com/srwiley/oksvg | BSD-3-Clause |
| github.com/srwiley/rasterx | BSD-3-Clause |
| github.com/yuin/goldmark | MIT |
| golang.org/x/image | BSD-3-Clause |
| golang.org/x/net | BSD-3-Clause |
| golang.org/x/sys | BSD-3-Clause |
| golang.org/x/text | BSD-3-Clause |

> The list covers the modules compiled into the application binary. Fyne's
> command-line build tooling (invoked only during packaging, never linked into
> Kalimate) pulls in additional modules, including one GPL-licensed font helper
> (`github.com/golang/freetype`); those are build-time tools and are not
> distributed as part of Kalimate.

To regenerate this inventory:

```sh
go list -deps -f '{{if .Module}}{{.Module.Path}}{{end}}' ./... | sort -u
```
