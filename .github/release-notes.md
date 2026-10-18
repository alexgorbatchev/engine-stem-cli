## Summary

- Inspect, encrypt, decrypt, and extract Engine DJ stem containers with `engine-stem-cli`.
- Prebuilt archives cover macOS and Linux on Intel and ARM64, with SHA-256 checksums.
- Encryption and decryption extract the key from a local Engine DJ Mach-O binary with a supported ARM64 initializer, without bundled application key bytes. Extraction also requires FFmpeg.
- Source separation is unfinished: `generate` currently encrypts its input without separating or encoding audio. Use `encrypt` with prepared multichannel M4A input.
