# journal-parser

## Compiling

```bash
sudo apt-get install libsystemd-dev
go build cmd/main.go -o journalparser
```

## Usage

```bash
journalparser [FLAGS]
Flags:
    -h, --help    Display this help message
    -t, --target <Directory or filename>  The target to parse
    -p, --partition <Number>  Max strings per file (no partition by default)
    --csv Export output to csv
    --json Export output to json
    --cli Export output to cli
```

