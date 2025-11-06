# mk-bkconv

A Go tool to convert backup files between the Mihon and Kotatsu Android apps.

---

<!-- Note: this project is educational and MIT-licensed. -->

> [!IMPORTANT]
> Educational purpose only — this tool is provided under the MIT license. It is not intended to help circumvent paywalls, license restrictions, or facilitate piracy. Use at your own risk and only with data you own or have permission to process.

## Table of contents

- [Status](#status)
- [Features](#features)
- [Requirements](#requirements)
- [Build](#build)
- [Usage](#usage)
- [Design notes](#design-notes)
- [Limitations & next steps](#limitations--next-steps)
- [Contributing](#contributing)
- [License](#license)

## Status

- Scaffolding implemented.
- Mihon -> Kotatsu conversion implemented for core fields (manga, chapters, categories).
- Kotatsu -> Mihon conversion implemented for core fields (basic mapping). More fields can be added.

## Features

- Convert Mihon backup (.tachibk — protobuf, optionally gzipped) to Kotatsu ZIP-of-JSON backup.
- Convert Kotatsu ZIP backup (JSON sections inside) to a minimal Mihon protobuf backup.
- Modular code (separate packages for Mihon, Kotatsu, conversion) and a simple CLI.

## Requirements

- Go 1.20+

## Build

The following commands work on PowerShell, Command Prompt, Bash, and other POSIX-like shells (replace paths as needed).

1. Clone the repository and change into the project directory:

```bash
git clone https://github.com/galpt/mk-bkconv.git
cd mk-bkconv
```

2. Ensure module dependencies are tidy and cached:

```bash
go mod tidy
```

3. Build the CLI:

```bash
go build ./cmd/mk-bkconv
```

This produces an executable named `mk-bkconv` (or `mk-bkconv.exe` on Windows) in the current directory.

Windows users: for faster iterative builds you can run the included `compile.bat` from the repository root — it runs `gofmt`, `go vet` (optional) and `go build` and pauses so you can read output.

Cross-compilation (optional): set `GOOS`/`GOARCH` environment variables before building, for example:

```bash
# Linux from Windows (example)
SET GOOS=linux
SET GOARCH=amd64
go build -o mk-bkconv-linux ./cmd/mk-bkconv
```

Or on Bash:

```bash
GOOS=linux GOARCH=amd64 go build -o mk-bkconv-linux ./cmd/mk-bkconv
```

## Usage

Two subcommands are available:

- `mihon-to-kotatsu` — convert a Mihon `.tachibk` backup to a Kotatsu ZIP.
- `kotatsu-to-mihon` — convert a Kotatsu ZIP backup to a Mihon `.tachibk` (basic mapping).

Examples (PowerShell):

```powershell
# Mihon -> Kotatsu
.\mk-bkconv.exe mihon-to-kotatsu -in C:\path\to\app.mihon_2025-11-01.tachibk -out C:\tmp\kotatsu_backup.zip

# Kotatsu -> Mihon
.\mk-bkconv.exe kotatsu-to-mihon -in C:\path\to\kotatsu_backup.zip -out C:\tmp\app.mihon_new.tachibk
```

> [!TIP]
> If you want deterministic outputs for testing, run the conversions on small sample backups first and inspect the resulting ZIP and JSON contents.

## Design notes

- Mihon backups are produced using Kotlin `kotlinx.serialization.protobuf` annotations (`@ProtoNumber`) and are usually gzipped. The tool detects gzip magic bytes and decodes accordingly.
- Kotatsu backups are ZIP files containing JSON arrays under named sections (e.g., `favourites`, `categories`, `history`).
- For an MVP I implemented a minimal protobuf wire reader/writer in `pkg/mihon` that handles the fields needed for basic migrations (varint, length-delimited strings, 32-bit floats for chapter numbers). This avoids requiring `protoc` and generated code during early development.
- For full fidelity and long-term robustness, reconstructing the `.proto` definitions from Mihon's Kotlin models and generating Go bindings via `protoc` is recommended.

### Data and privacy

> [!WARNING]
> Backup files contain private user data (reading history, bookmarks, possibly preferences). Do not share backups or processed outputs unless you have explicit permission.

> [!CAUTION]
> Converting large backups may use significant memory. The current implementation decodes into memory for simplicity; for very large backups we should add streaming and per-entry processing.

## Limitations & next steps

1. The current protobuf implementation covers core fields (manga, chapters, categories). Several Mihon sections (preferences, sources, extension repos, trackers) are not yet mapped.
2. Suggested improvement: automatically generate `.proto` files from Mihon Kotlin `@ProtoNumber` annotations and use `protoc` to produce canonical Go types (recommended for correctness and maintainability).
3. Add unit tests for round-trip conversions and more comprehensive field mapping.

## Contributing

Contributions welcome. Please open issues or PRs. If you want me to generate `.proto` files from the Kotlin sources and wire up `protoc` generation, tell me and I will implement that next.

## License

MIT. See the `LICENSE` file.
