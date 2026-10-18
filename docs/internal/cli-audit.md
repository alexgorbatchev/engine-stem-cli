---
created_on: 2026-10-18 09:53
last_modified: 2026-10-18 09:53
status: current
---

# CLI best-practices audit

This audit compares `engine-stem-cli` with the CLI best-practices and Go engineering skills. Findings are grounded in the command implementations and local test runs. It is an audit, not a claim of full conformance.

## Findings

| Priority | Finding | Evidence and impact |
| :--- | :--- | :--- |
| High | Source separation is unfinished | `internal/stems/generate.go` opens and closes an ONNX session, never calls `Separate`, and calls `Encrypt` directly. It does not encode audio or apply the bitrate. `generate --dry-run` describes steps that do not execute. |
| High | Extraction dry runs modify temporary files | `internal/cli/extract.go` decrypts encrypted input before checking `extractDryRun`. Existing `TestExtractCmdDryRunEncrypted` exercises this path. |
| Medium | Required agent skill is absent | No `skills/engine-stem-cli/SKILL.md`, top-level `skill` command, agent help alert, or interface-drift tests exist. |
| Medium | Domain commands are verb-first | `decrypt`, `encrypt`, `extract`, and `generate` are top-level commands rather than actions under a subject. Changing this interface requires a separate migration. |
| Medium | Help does not meet the tree and agent contracts | `internal/cli/root.go` uses Cobra's default help. The required `cobra-help-tree/v2` integration, argument catalog, width trimming, and compact agent help are absent. |
| Medium | Home paths are not folded | `internal/cli/output.go` prints message and metadata paths unchanged. Commands and verbose workflows also print paths directly. No shared home-path formatter exists. |
| Medium | Output errors are discarded | Output helpers ignore formatted-write errors; table rendering discards `Append` and `Render` errors. Root version output also discards write failures. |
| Medium | Coverage misses the project minimum | Local `just ci` reports CLI 81.8%, inference 27.1%, MP4 87.3%, and stems 88.6%. The 90% policy is not enforced by CI. |
| Low | Existing formatting drift | `gofmt -l cmd internal` reports `internal/inference/onnx.go`. |

## Conforming areas

- Cobra parses arguments and flags.
- `AGENT=1`, `true`, and `yes` select compact status and metadata output, with case and surrounding whitespace normalized.
- Human metadata uses the tablewriter library; agent metadata uses key-value lines.
- `Justfile` exposes `run`, `run-ai`, `test`, `build`, and static-check recipes.
- Builds go under ignored `bin/`; MIT licensing credits Alex Gorbatchev.
- Release automation builds native macOS and Linux archives for amd64 and arm64, checks their runtime version against the tag, and publishes checksums only after all verification jobs succeed.
- README installation uses GitHub release archives and documents unfinished generation and extraction dry-run side effects.
- Application key bytes and the fixed XOR mask are removed from production code and tests. Extraction reads a supported ARM64 Mach-O initializer tied to the imported AES function. Test fixtures carry synthetic values.
- The crypto package reaches 91.2% statement coverage and the shared binary fixture reaches 93.3% in local race-enabled tests.

## Verification

- Baseline `just test`: passed.
- Rename regression: failed with the old command name, passed with `engine-stem-cli`, and failed again when the root command name was temporarily reverted.
- `just ci`: module hygiene, vet, and race-enabled tests passed; coverage gaps are listed above.
- `VERSION=v1.0.0 just release-package`: native macOS ARM64 archive built, runtime version matched, and archive members included the binary, README, and license.
- Key extraction regression: two synthetic binaries supply different keys; both fail with the old scanner, pass with extraction, and fail when extraction is temporarily disabled. Tests also reject unsupported formats, architectures, imports, branches, and missing key references, and cover universal Mach-O selection and cache isolation.
- Real installed `OfflineAnalyzer`: extracted key decrypts a local Engine DJ stem container; FFmpeg decodes the first three seconds without errors. Key bytes and decrypted audio remain outside Git.

GitHub workflow results and published assets require verification after publication. Local checks do not establish that other platforms pass.
