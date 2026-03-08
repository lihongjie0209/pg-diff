# pg-diff
`pg-diff` is an advanced PostgreSQL schema comparison execution tool built in Go. It dynamically evaluates source and target database schemas and produces precise Data Definition Language (DDL) migration scripts capable of gracefully bridging configuration gaps.

## Features
- **Topological Diff Generation**: Seamlessly compares Extensions, Custom types (Enums), Sequences, Tables, Columns, Constraints, Indices, Views, Functions, Triggers, RLS Policies, Privileges, and Object Comments.
- **Intelligent Dependency Resolution**: When querying explicit objects (e.g. `--object-types="table,view"`), `pg-diff` leverages a Breadth-First Search (BFS) heuristic graph algorithm to automatically traverse and include reliant SQL dependencies (Enums, default Sequences, Foreign Key relations).
- **Flyway Native Versioning**: Outputs safely into your existing API directory structure by analyzing numbering formats and sequentially staging the correct incrementally stamped migration script.
- **Dry-run Confines**: Supports native CLI dry-run previews validating final target syntax blocks explicitly prior to impacting local disks.

## Usage

```bash
# Minimal execution targeting purely console output
pg-diff --source "postgres://user:pass@host:5432/source?sslmode=disable" \
        --target "postgres://user:pass@host:5432/target?sslmode=disable"

# Scoped execution generating a flyway-sequenced output with preview (dry-run)
pg-diff --source "postgres://user:pass@host:5432/source?sslmode=disable" \
        --target "postgres://user:pass@host:5432/target?sslmode=disable" \
        --object-types "table,view,function" \
        --flyway-dir "./migrations" \
        --schema "public" \
        --dry-run
```

## Automating Releases
This repository incorporates automated GitHub Action CI/CD deployment logic bound tightly to Git tracking tags. Drafting a local standard version flag (`git tag v1.0.0` => `git push origin v1.0.0`) inherently queues cross-compilation matrix runners generating `.exe` pipelines alongside raw Unix builds for automatic Release.
