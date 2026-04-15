# skl

Manage Claude Code skills as named bundles. Source-of-truth library lives in
`~/.config/skl/library/`; `skl load <bundle>` materializes a bundle into
`~/.skills/`, `skl unload` removes it.

## Why

Claude scans every directory in `~/.skills/`. With 70+ skills installed, that's
noisy and slow. `skl` lets you keep a curated library and load only the slice
you need for the current task: `gstack`, `writing`, `dev`, etc.

## Install

```sh
make install      # builds and copies to ~/bin/skl
```

Requires `fzf` for interactive picking and `git` for sync.

## Quick start

```sh
skl import                                  # seed library from current ~/.skills/
skl bundle create dev cso ts-style-review   # group skills into a bundle
skl load dev                                # copy the bundle into ~/.skills/
skl status                                  # what's currently loaded
skl unload dev                              # remove the bundle
skl unload                                  # fzf picker over loaded bundles
skl ls                                      # list all bundles
skl ls --skills                             # list every skill in the library
skl install https://github.com/obra/superpowers --bundle superpowers
skl remote git@github.com:you/skl-library.git
skl sync                                    # commit + pull --rebase + push
```

## Layout

```
~/.config/skl/
  config.yaml                    # remote, default bundles
  library/
    bundles.yaml                 # named groups
    skills/<skill>/...           # local skill folders (have SKILL.md)
    external/<repo>/<skill>/...  # cloned third-party skills

~/.local/state/skl/
  state.json                     # what's currently loaded, by which bundle
  state.lock                     # flock for atomic mutations

~/.skills/                       # the live target Claude reads
  <skill>/...                    # only what skl has loaded
  .system/                       # never touched
```

## Commands

| Command | Purpose |
|---|---|
| `skl ls` | List bundles. `--skills` lists individual skills. |
| `skl status` | Show what's loaded, what's untracked, what's drifted. |
| `skl load [bundle...]` | Load bundles. fzf picker if no args. `--skill <name>` for individual skills. |
| `skl unload [bundle...]` | Unload bundles. fzf picker if no args. `--all`, `--skill`. |
| `skl bundle create <name> <skill...>` | Create or replace a bundle. |
| `skl bundle add <name> <skill...>` | Append skills to a bundle. |
| `skl bundle remove <name> <skill...>` | Drop skills from a bundle. |
| `skl bundle rm <name>` | Delete a bundle. |
| `skl import` | Copy current `~/.skills/` into the library. |
| `skl push <skill>` | Copy a live `~/.skills/<skill>` back into the library. |
| `skl install <git-url>` | Clone a remote skill repo into `library/external/`. `--bundle <name>` to group. |
| `skl remote [url]` | Set or show the git remote for the library. |
| `skl sync` | Commit, `pull --rebase`, push. |
| `skl config` | Show config and library paths. |

## Bundle semantics

A skill can belong to multiple bundles. Loading two bundles that share a skill
loads the skill once and records both claims. Unloading one bundle keeps the
skill on disk if another loaded bundle still claims it.

Edits to `~/.skills/<skill>` do not propagate back to the library. Run
`skl push <skill>` to capture them.

Dot-prefixed entries in `~/.skills/` (`.system`, `.llm`, `.DS_Store`, …) are
never touched.
