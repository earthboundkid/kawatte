# Kawatte [![GoDoc](https://godoc.org/github.com/earthboundkid/kawatte?status.svg)](https://godoc.org/github.com/earthboundkid/kawatte) [![Go Report Card](https://goreportcard.com/badge/github.com/earthboundkid/kawatte)](https://goreportcard.com/report/github.com/earthboundkid/kawatte)

Kawatte recursively walks the file tree and finds and replaces the patterns found in a substitution file.

## Installation

First install [Go](http://golang.org).

If you just want to install the binary to your current directory and don't care about the source code, run

```bash
GOBIN="$(pwd)" go install github.com/earthboundkid/kawatte@latest
```

## Screenshots

```
kawatte - (devel)

Kawatte recursively walks the file tree and finds and replaces the patterns found in a substitution file.

Usage:

	kawatte [options]

Options:
  -dir directory
    	path to the starting directory (default ".")
  -dry-run
    	just print the names of files that would be modified
  -exclude glob
    	glob matching files to exclude (default .*)
  -exclude-dir glob
    	glob matching directories to exclude (default .*)
  -match glob
    	glob matching files to include (default *)
  -match-dir glob
    	glob matching directories to include (default *)
  -pat file
    	path to the CSV file containing substitution patterns
  -v	short alias for -version
  -verbose
    	log debug output
  -version
    	print version information and exit
```
