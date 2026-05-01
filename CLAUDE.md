# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`skl` is a Go CLI that manages the user's Claude Code skills directory (`~/.skills/`) as folder bundles. Skills are stored canonically in `~/.config/skl/library/` and copied into `~/.skills/` on `skl load <bundle>`. Removed on `skl unload`.

## Build & Run

```sh
make build      # builds ./skl with version stamped
make install    # builds and copies to ~/bin/skl
```

Module name is `skl` (plain, not URL).

## Architecture

**CLI layer** (`cmd/`): one file per command. `cmd/root.go` wires Cobra and the custom grouped help template (Inspect / Load / Library / Sync / Other). `cmd/bundle.go` is a parent for `bundle_*.go` subcommands. `ErrCancelled` exits cleanly when the user aborts an fzf prompt.

**Internal packages** (`internal/`):
- `config/` — YAML config at `~/.config/skl/config.yaml`. Auto-creates with header on first read.
- `library/` — Source-of-truth filesystem at `~/.config/skl/library/`. Discovers folder bundles by walking skill directories. `skills/<skill>` is legacy unbundled storage; `external/<repo>/<skill>` is a namespaced external bundle.
- `live/` — `~/.skills/` filesystem. `CopySkill` (recursive copy with rollback), `RemoveSkill` (refuses dot names), `LoadedDirs`. Always skips dot-prefixed entries.
- `state/` — JSON state at `~/.local/state/skl/state.json`. Flock-locked via `syscall.Flock()` (grove pattern). Tracks per-skill bundle claims for reference-counted unload.
- `bundle/` — Pure functions: `PlanLoad`, `PlanUnload`. No I/O. Resolves which skills to copy/skip and which to remove vs keep on unload.
- `picker/` — Thin `fzf` wrapper. Returns `ErrCancelled` on exit code 130.
- `gitlib/` — Thin shell-out to git for sync (`init`, `add`/`commit`, `pull --rebase`, `push`, `clone`).
- `style/` — Shared lipgloss colors and helpers.

**Data flow**: library = source of truth → `skl load <bundle>` resolves the bundle folder's direct skills, copies skill trees into `~/.skills/`, records each (skill, bundle) claim in state. `skl unload <bundle>` reads state, decrements bundle claims; only removes a skill from disk when no claim remains.

## Key Invariants

- **Never touch dot-prefixed entries in `~/.skills/`** — `live.guardDirName` enforces.
- **State mutations require `mgr.Lock()` / `defer mgr.Unlock()`** — same flock pattern as grove.
- **Atomic writes everywhere** — tmp + rename for `state.json`, legacy `bundles.yaml`, `config.yaml`.
- **Per-bundle atomicity on load** — failure mid-bundle rolls back that bundle's copies; other bundles unaffected.
- **Reference-counted unload** — a skill held by N bundles is only removed when the Nth bundle is unloaded.
- **External skills are namespaced** — `external/<repo>/<skill>` → ID `external/<repo>/<skill>` to avoid collisions with local bundle paths.

## Two helpers worth flagging

- `internal/live/live.go:copyTree` and `cmd/copy_helper.go:copyDir` are sibling implementations doing the same recursive copy. The `live` one writes into `~/.skills/`; the `cmd` one writes into the library (`import`, `push`). Kept separate so the strict dot-name guard lives only on the `live` side.
- `cmd/load.go:applyLoadPlan` performs the actual copy + state mutation in lockstep — copy first, claim second, so partial state is impossible without a partial copy preceding it (which the rollback handles).
