> [!WARNING]
> This public repository does not include Engine DJ encryption keys, proprietary Engine DJ source code, or Engine DJ application binaries. Encryption keys are extracted at runtime from your own local Engine DJ installation. Test fixtures use synthetic key values.

`engine-stem-cli` helps DJs inspect Engine DJ stem containers, decrypt them to multichannel M4A, and extract individual stereo parts.

# What It Does

- **Inspect:** Show metadata and encryption status, including JSON output.
- **Decrypt:** Convert encrypted stem containers to multichannel M4A.
- **Encrypt:** Package prepared multichannel M4A audio as Engine DJ stems.
- **Extract:** Write vocals, drums, bass, and other stereo tracks with FFmpeg.

# How It Works

- Provide a `.stems` or prepared multichannel `.m4a` file.
- Inspect its metadata, encrypt or decrypt its audio, or extract four stereo pairs.
- Choose an output path explicitly or use the command's default filename.

# How it Really Works

- Encryption and decryption read the key initializer from a local Engine DJ Mach-O application binary with an ARM64 slice. Unsupported layouts return an error. Set `ENGINE_DJ_APP_PATH` to select a binary outside the default macOS installation paths, including when running on Linux or Intel macOS.
- Extraction assigns channels 0–1 to vocals, 2–3 to drums, 4–5 to bass, and 6–7 to other. It overwrites existing extracted files.
- Encrypted extraction creates and removes a temporary decrypted file, including with `--dry-run`; that option only skips FFmpeg execution.
- `AGENT=1` selects compact status and metadata output. Progress and results share stdout; verbose subprocess diagnostics can also appear on stderr. Failures exit with status 1.
- Source separation is unfinished. `generate` currently encrypts its input without separating audio or applying `--bitrate`; use `encrypt` with prepared multichannel M4A audio.
- Linux archives target Ubuntu 24.04. macOS archives are unsigned.

# Installation

Download the archive for your platform from the [latest release](https://github.com/alexgorbatchev/engine-stem-cli/releases/latest), replacing `vX.Y.Z` with its tag.

```bash
# macOS (Apple Silicon)
mkdir -p ~/.local/bin
curl -fL https://github.com/alexgorbatchev/engine-stem-cli/releases/download/vX.Y.Z/engine-stem-cli_vX.Y.Z_darwin_arm64.tar.gz | tar -xz -C ~/.local/bin engine-stem-cli
```

# Setup

- [Engine DJ Desktop](https://enginedj.com/): Required for encryption and decryption, including encrypted extraction. Default binary locations are inside `/Applications/Engine DJ.app/Contents/MacOS/`.
- [FFmpeg](https://ffmpeg.org/download.html): Required for `extract`; make `ffmpeg` available on `PATH`.

# Quick Start

```bash
engine-stem-cli info track.stems
AGENT=1 engine-stem-cli info track.stems
engine-stem-cli info track.stems --json
engine-stem-cli decrypt track.stems
engine-stem-cli encrypt track.m4a
engine-stem-cli extract track.stems -o ./stems-output
engine-stem-cli --version
```

Sample Output (`--version` for release `v1.0.0`):

```text
v1.0.0
```

# Options & Flags

| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--verbose` | — | `false` | Enable diagnostic logging |
| `--quiet` | `-q` | `false` | Suppress command status output |
| `--version` | `-v` | `false` | Print version (root command) |
| `--help` | `-h` | `false` | Show command help |

`engine-stem-cli info <file>`

| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--json` | — | `false` | Print metadata as JSON |

`engine-stem-cli decrypt <input.stems> [output.m4a]` and `engine-stem-cli encrypt <input.m4a> [output.stems]`

| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--output <path>` | `-o` | `""` | Override positional output; otherwise replace the recognized input extension or append the output extension |
| `--force` | `-f` | `false` | Overwrite output |

`engine-stem-cli extract <file> [output-dir]`

| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--output-dir <path>` | `-o` | `""` | Override positional directory; otherwise `<input-basename>-stems` in working directory |
| `--format <format>` | — | `wav` | Write `wav` or `m4a` tracks |
| `--dry-run` | — | `false` | Print FFmpeg invocation after input inspection and any decryption |

`engine-stem-cli generate <input-track> [output.stems]` (unfinished separation)

| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--output <path>` | `-o` | `""` | Override positional output; otherwise replace input extension with `.stems` |
| `--model <path>` | `-m` | `""` | Open ONNX model; requires ONNX Runtime 1.28.0 shared library; inference is unfinished |
| `--bitrate <rate>` | `-b` | `640k` | Accepted but not applied |
| `--force` | `-f` | `false` | Overwrite output |
| `--dry-run` | — | `false` | Print intended pipeline without execution |

# License

[MIT License](LICENSE), copyright 2026 Alex Gorbatchev.
