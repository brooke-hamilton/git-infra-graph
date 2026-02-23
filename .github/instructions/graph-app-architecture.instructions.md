---
applyTo: "**"
description: "Use when working on Git object storage, graph data modeling, traversal patterns, or architecture documentation. Covers Git internals, graph database design, and documentation conventions for the grif project."
---

# Graph Application Architecture

## Git Internals

This project stores graph data in Git's object database. Understand these internals:

- **Object model**: content-addressable storage of blobs, trees, commits, and tags as a directed acyclic graph
- **Refs**: branches as pointers to commits, HEAD, tags (lightweight and annotated), symbolic refs, and refspecs
- **Object database**: SHA-1/SHA-256 hashing, packfiles, loose objects, and garbage collection
- **Plumbing commands** (for debugging): `git cat-file`, `git hash-object`, `git update-ref`, `git ls-tree`, `git rev-parse`
- **Merge internals**: three-way merge, recursive strategy, conflict resolution

When implementing features, reason about how operations map to Git objects — a node is a blob, a graph snapshot is a tree, a graph version is a commit.

## Graph Data Modeling

- Design property graphs with clear node labels, relationship types, and property schemas
- Keep traversal patterns in mind: shortest path, breadth-first, depth-first, pattern matching
- Use appropriate index strategies (composite, full-text) based on query patterns
- Profile queries for cardinality estimation and performance tuning
- Prefer graph storage when relationships are first-class; use relational or document stores when entities are independent

## Architecture Documentation

- Write architecture documents in Markdown
- Use Mermaid diagrams (fenced `mermaid` code blocks) for flowcharts, sequence diagrams, ER diagrams, C4 models, and deployment views
- Use ASCII art diagrams for simple visuals, inline docs, READMEs, or terminal-friendly contexts
- Structure architecture decision records (ADRs) with context, decision, and consequences sections
- Choose the diagram type for the audience — Mermaid for rich detail, ASCII art for simplicity

## Design Principles

- Prefer simplicity over cleverness — choose the simplest solution that meets requirements
- When multiple valid approaches exist, present trade-offs rather than prescribing a single answer
- Stay within the project's technology constraints (Go, go-git, Git object database)
- Provide working code examples rather than pseudocode
