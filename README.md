# Totion

Totion is a small terminal note app for Markdown notes. It stores notes in
`~/.totion` and uses Bubble Tea for the text interface.

## Features

- Create Markdown notes from the terminal.
- Open and filter existing notes.
- Save notes with `Ctrl+S`.
- Keep notes as plain `.md` files in a local vault.

## Requirements

- Go 1.25 or newer.

No Docker setup is required. The app is built and run with the Go toolchain.

## Usage

```sh
go run .
```

Or build a local binary:

```sh
make build
./totion
```

## Keyboard Shortcuts

- `Ctrl+N`: create a note
- `Ctrl+L`: open the notes list
- `Ctrl+S`: save the current note
- `Esc`: go back or close the current view
- `Q`: quit

## Development

```sh
make fmt
make build
```

The repository intentionally avoids committing generated binaries and Docker
artifacts. Keep the source runnable with `go run .` and keep formatting aligned
with `gofmt`.
