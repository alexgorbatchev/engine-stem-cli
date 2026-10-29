# AGENTS.md

Guidelines and non-negotiable rules for AI coding agents working on `engine-stem-cli`.

## Mandatory Rule

**DO NOT USE CLI TOOLS TO CREATE OR EDIT SOURCE FILES.**
Always use the agent harness editing tools (`write`, `edit`) to create or modify source code, tests, documentation, or configuration files.

---

## Go Engineering Conventions

1. **Package Layout**:
   - `cmd/engine-stem-cli/`: Application entry point.
   - `internal/cli/`: Cobra CLI commands, flags, and UI output formatting.
   - `internal/crypto/`: AES-128-ECB sample payload encryption/decryption and PKCS#7 padding.
   - `internal/mp4/`: MP4 container atom parsing, box discovery, and sample/chunk offset calculations.
   - `internal/stems/`: High-level encryption/decryption container rebuild workflows.

2. **Tooling & Build System**:
   - Always use `Justfile` tasks for building, testing, and linting:
     - `just build`: Compiles binary into `./bin/engine-stem-cli`.
     - `just setup`: Downloads module dependencies.
     - `just test`: Runs all unit tests (`go test -v ./...`).
     - `just vet`: Runs Go static analysis (`go vet ./...`).
     - `just bench`: Runs performance benchmarks (`go test -bench=. ./...`).
     - `just ci`: Checks module hygiene, runs vet, and runs race-enabled tests with coverage.
     - `VERSION=vX.Y.Z just release-package`: Builds a native archive under `bin/releases/` and verifies its runtime version.
   - Binaries MUST be written to `./bin/` and MUST NEVER be committed to Git (enforced via `.gitignore`).

3. **Testing Standards**:
   - Maintain minimum **90% line statement coverage** across all packages.
   - Use table-driven tests (`tests := []struct{ ... }`) for function unit tests.
   - Mark test setup helpers with `t.Helper()`.

4. **Error Handling & Code Style**:
   - Wrap errors with context using `%w` (`fmt.Errorf("doing x: %w", err)`).
   - Never discard runtime errors with `_`.
   - Variable names match scope distance (short for tight loops, descriptive for package/struct scope).

---

## Repository Branching Conventions

- **`main`**: Clean, production-ready Go CLI application codebase, `Justfile`, unit tests, and evergreen user documentation.
- **`research`**: Exploratory research scripts (Python, TypeScript), reverse-engineering disassembly notes, and baseline measurements.
- Start every unit of work from the latest revision: `git fetch`, then bring the local target branch level with its remote branch (`git merge --ff-only origin/main` in the main checkout for `main`; `git fetch origin research:research` for `research`). If it cannot fast-forward, the branch holds commits from a failed push; rebase them onto the remote branch and push before anything else. Never begin work on a stale branch.
- Make every change in a linked worktree under `.workspaces/<name>` on a new branch created from that up-to-date target branch, `main` or `research` (`git worktree add -b <branch> .workspaces/<name> main`). Never edit the main checkout directly; other sessions share it.
- Validate and commit in the worktree, then fast-forward the target branch from the main checkout (`git merge --ff-only <branch>`). Rebase the work branch first if the target moved. Remove the merged worktree and branch afterward. Commits and merges remain local; publish only through an explicit push or workspace synchronization.
- Never install automatic post-commit or post-merge push hooks. Keep commits local; global dotfiles Git hooks enforce secret scanning independently of repository hooks.
- Never amend, rebase, or force-push commits already published upstream; fix mistakes with a follow-up change. Inspect staged changes for private hostnames before committing. A commit or merge does not authorize an automatic push.

---

## Findings & Specifications

Keep factual findings grounded strictly in verified source code and test measurements. Do not add unconfirmed speculation to documentation.

## Releases

GitHub Actions verifies native macOS and Linux builds on amd64 and arm64. ONNX uses CGO, so release builds run on native runners. Archives contain the binary, README, and license; releases include SHA-256 checksums.

Maintain `.github/release-notes.md` before publishing a version tag. Never publish releases, tags, packages, or production deployments without explicit user authorization.
