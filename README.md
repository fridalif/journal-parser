# journal-parser

## Description

Tool for recursive parsing Linux journals with export in different file formats

## Used Libraries

- [Velocidex Journalctl](github.com/Velocidex/go-journalctl);
- [Velocidex GoNTFS](www.velocidex.com/golang/go-ntfs);
- [Progressbar](github.com/schollz/progressbar);
- [SQLite Driver](github.com/mattn/go-sqlite3).

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
    -t, --target <Directory or filename>  The target to parse (this argument can be repeated)
    -p, --partition <Number>  Max strings per file (min 0, no partition by default)
    -mb, --max-batch <Number> Max number of butch for SQL insert (max 1000, min 1, default 1000)
    --csv Export output to csv
    --json Export output to json
    --cli Export output to cli
```

## Examples

```bash
# Parsing journal in default mode (no limits for RAM) with export in csv only
journalparser -t /var/log/journal/ --csv

# Parsing many journal directories with partition (limits for count of entries in memory and output)
# with export in csv, json and cli
journalparser -t /var/log/journal/ -t /var/log/journal2/ -p 10000 --csv --json --cli

# Parsing journal file with max batch (in sql) limit with export in cli
journalparser -t /var/log/journal/system.journal -mb 500 --cli
```