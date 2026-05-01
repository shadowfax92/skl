<div align="center">

# 🎒 skl

**A loadout manager for Claude Code skills.**

*Bundle them, load them, share them. Your `~/.skills/` without the noise.*

</div>

Claude scans every folder in `~/.skills/`. Once you've got 70+ skills installed, most are irrelevant to any given task — they burn context window and slow startup. `skl` keeps the canonical library separate from the live directory: curate skills into folder bundles (`dev`, `writing`, `external/gstack`), then load and unload them per-project.

- 🎒 **Folder bundles** — group skills by directory and load only what you need
- ⚡ **Load / unload** — `skl load dev` copies the bundle into `~/.skills/`, `skl unload` removes it
- 🔍 **fzf everywhere** — any command with no args drops into an interactive picker
- 📦 **Third-party packs** — `skl install <git-url>` pulls shared skill collections (e.g. [obra/superpowers](https://github.com/obra/superpowers))
- 🏷️ **Prefix or namespace** — disambiguate third-party skills as `supa-brainstorming` or `external/superpowers/brainstorming`
- 🔄 **Git sync** — `skl sync` backs up the library to any git remote
- 🛡️ **Safe by default** — dot-entries in `~/.skills/` (`.system`, `.DS_Store`) never touched; symlinks skipped during copy
- 🔁 **Reference-counted** — a skill shared by two loaded bundles only leaves disk when both are unloaded

---

## Install

Requires Go 1.24+, [fzf](https://github.com/junegunn/fzf), and git.

```sh
git clone https://github.com/shadowfax92/skl
cd skl
make install    # builds and copies to ~/bin/
```

Make sure `~/bin` is on your `PATH`.

## Quick Start

```sh
skl import                              # seed library from current ~/.skills/
skl prune --untracked                   # wipe the old uncategorized dregs
skl bundle create dev
mv ~/.config/skl/library/skills/cso ~/.config/skl/library/dev/cso

# use them
skl load dev                            # ~/.skills/ gets only dev's skills
skl ls                                  # what bundles exist
skl status                              # what's loaded right now
skl unload                              # fzf-pick a loaded bundle to remove

# import a third-party pack
skl install https://github.com/obra/superpowers --subdir skills
skl load external/superpowers
```

## Commands

### Inspect

```sh
skl ls                          # list bundles (aliases: list)
skl ls --skills                 # list every skill with bundle membership
skl status                      # what's in ~/.skills/ right now (aliases: st)
skl config                      # show config and library paths
```

### Load & unload

```sh
skl load [bundle...]                    # load bundles (fzf if no args)
skl load --skill foo --skill bar        # load individual skills
skl unload [bundle...]                  # unload bundles (fzf if no args)
skl unload --all                        # unload everything skl loaded
skl prune                               # fzf-pick skills to wipe from ~/.skills/ (aliases: rm)
skl prune foo bar                       # remove specific skills by name
skl prune --untracked                   # remove skills not loaded by skl
skl prune --all                         # nuke everything in ~/.skills/
```

### Library management

```sh
skl import                              # copy ~/.skills/ → library/
skl push <skill>                        # capture edits from live back into library
skl install <git-url | path>            # import third-party skills (see below)
skl bundle create <name>                # create a folder bundle
skl bundle rm <name>                    # delete an empty folder bundle
```

### Sync

```sh
skl remote <url>                # set the library's git remote
skl sync                        # commit + pull --rebase + push
```

## Installing third-party skills

Two modes, picked by whether you pass `--prefix`:

```sh
# Namespaced — skills live in library/external/superpowers/
# Load with  skl load external/superpowers
skl install https://github.com/obra/superpowers --subdir skills

# Flat prefix — skills copied into library/skills/supa-<skill>/
# Show up as unbundled legacy skills
skl install https://github.com/obra/superpowers --subdir skills --prefix supa

# Local paths work too
skl install /Users/you/code/my-skills/skills --prefix my
```

Flags:

| Flag | Purpose |
|------|---------|
| `--subdir <path>` | Many repos nest skills under `skills/` — pass the subfolder to scan |
| `--prefix <name>` | Install flat as `library/skills/<prefix>-<skill>/` |
| `--bundle <name>` | Deprecated; folder bundles are directory-based |
| `--name <name>` | Override the namespace dir name (namespaced mode only) |
| `--force` | Overwrite existing skills or namespaces on re-install |

## How it works

Three filesystem layers:

1. **Library** (`~/.config/skl/library/`) — source of truth
   - `<bundle>/<skill>/…` — folder bundles
   - `skills/<skill>/…` — legacy unbundled skills
   - `external/<ns>/<skill>/…` — namespaced third-party folder bundles
2. **Live** (`~/.skills/`) — the directory Claude reads. `skl` copies skill trees in and out.
3. **State** (`~/.local/state/skl/state.json`) — which skills are loaded and by which bundle. Flock-protected for atomic mutations.

Each directory containing direct skill subdirectories is a loadable bundle. Loading two bundles that both resolve to the same skill keeps reference-counted claims in state; unloading one bundle keeps the skill on disk until the last claim is gone.

Dot-prefixed entries in `~/.skills/` (`.system`, `.llm`, `.DS_Store`, etc.) are never touched. Dot-prefixed entries in the library, including nested `.git` directories, are skipped during skill discovery. Symlinks inside skills are skipped during copy — an untrusted third-party repo can't slip a symlink to `/etc/passwd` into your `~/.skills/`.

## Config

`~/.config/skl/config.yaml` (created on first run):

```yaml
sync:
  remote: git@github.com:you/skl-library.git
default_bundles: []
```

Most users never edit this directly — `skl remote <url>` sets `sync.remote` for you.

## Typical workflow

```sh
# One-time onboarding (you already have stuff in ~/.skills/)
skl import
skl prune --untracked
skl bundle create gstack
mv ~/.config/skl/library/skills/* ~/.config/skl/library/gstack/

# Day to day
skl load gstack      # switch to backend work
skl unload           # fzf-pick to clear when done
skl load writing     # switch to marketing work

# Occasionally
skl install <git-url> --subdir skills
skl push <skill>     # capture edits you made to a live skill
skl sync             # back up the library
```

---

> Personal tool built for my own workflow. Fork and adapt.
