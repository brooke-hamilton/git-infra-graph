# grif Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-02-13

## Active Technologies
- Go 1.26.0 (per `go.mod`) + `go-git` (`github.com/go-git/go-git/v5`) — pure-Go Git implementation for all object store operations (blobs, trees, commits, refs) (002-graph-put-get-delete)
- Git object database only (blobs, trees, commits, refs); per-graph index file (`.git/infra-index-<name>`) for staging (002-graph-put-get-delete)

- Go (version TBD in `go.mod`; target Go 1.22+) + `go-git` (`github.com/go-git/go-git/v5`) — pure-Go Git implementation; no runtime `git` binary required (see [research.md](research.md) R1) (001-graph-init-list)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go (version TBD in `go.mod`; target Go 1.22+)

## Code Style

Go (version TBD in `go.mod`; target Go 1.22+): Follow standard conventions

## Recent Changes
- 002-graph-put-get-delete: Added Go 1.26.0 (per `go.mod`) + `go-git` (`github.com/go-git/go-git/v5`) — pure-Go Git implementation for all object store operations (blobs, trees, commits, refs)

- 001-graph-init-list: Added Go (version TBD in `go.mod`; target Go 1.22+) + `go-git` (`github.com/go-git/go-git/v5`) — pure-Go Git implementation; no runtime `git` binary required (see [research.md](research.md) R1)

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
