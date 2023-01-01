# Cheat-sheet Command Line Tool

This command line tool is a simple wrapper of `tldr` that add additional function of adding && editing personal cheat-sheet.

It stores personal cheat-sheet at `$HOME/.cheat-sheet` directory. And Every time
when you find a cheat-sheet, it will first look at your personal cheat-sheets in the `$HOME/.cheat-sheet` directory. If it can't find it, it will then call `tldr` to find it.

## Install

```bash
go install github.com/yz-1209/cheat-sheet-tool@latest
```

## Usage

Usage is quite like `tldr`:
```bash
# List help info
cs -h

# Print openssl cheat-sheet
cs openssl

# Edit openssl cheat-sheet
cs -e openssl

```