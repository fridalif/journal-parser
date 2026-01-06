# journal-parser v1.1.0

## Description

Crossplatform tool for recursive parsing Linux journals directories and files with export in different file formats

## Used Libraries

- [Velocidex Journalctl](github.com/Velocidex/go-journalctl);
- [Velocidex GoNTFS](www.velocidex.com/golang/go-ntfs);
- [Progressbar](github.com/schollz/progressbar);
- [SQLite Driver](github.com/mattn/go-sqlite3).

## Compiling

```bash
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
    -o, --output <Directory> Directory for output
    --csv Export output to csv
    --json Export output to json
    --cli Export output to cli
    -fo, --files-offset <Number> Use this for skip files which you already parsed
    -fl, --files-limit <Number> Use this for limit files which you want parse
    -dbn, --db-name <Path> Name of existing database if needs append data in old DB
    -sp, --skip-parsing - Flag for skipping parsing (only export)
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