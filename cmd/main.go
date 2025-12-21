package main

import (
	"C"
	"fmt"
	journalparser "journal-parser/pkg/journal-parser"
	"os"
	"strconv"
	"strings"
	"time"
)

func bannerOutput() {
	fmt.Println(`------------------------------------------------------------------------------------------------`)
	fmt.Println(`      @                                              @@@@@@                                     `)
	fmt.Println(`      @  @@@@  @    @ @@@@@  @    @   @@   @         @     @   @@   @@@@@   @@@@  @@@@@@ @@@@@  `)
	fmt.Println(`      @ @    @ @    @ @    @ @@   @  @  @  @         @     @  @  @  @    @ @      @      @    @ `)
	fmt.Println(`      @ @    @ @    @ @    @ @ @  @ @    @ @         @@@@@@  @    @ @    @  @@@@  @@@@@  @    @ `)
	fmt.Println(`@     @ @    @ @    @ @@@@@  @  @ @ @@@@@@ @         @       @@@@@@ @@@@@       @ @      @@@@@  `)
	fmt.Println(`@     @ @    @ @    @ @   @  @   @@ @    @ @         @       @    @ @   @  @    @ @      @   @  `)
	fmt.Println(` @@@@@   @@@@   @@@@  @    @ @    @ @    @ @@@@@@    @       @    @ @    @  @@@@  @@@@@@ @    @ `)
	fmt.Println(`------------------------------------------------------------------------------------------------`)
}

func printHelpMessage() {
	fmt.Println("Usage: journalparser [FLAGS]")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  	-h, --help    Display this help message")
	fmt.Println("  	-t, --target <Directory or filename>  The target to parse (this argument can be repeated)")
	fmt.Println("  	-p, --partition <Number>  Max strings per file (min 0, no partition by default)")
	fmt.Println("	-mb, --max-batch <Number> Max number of butch for SQL insert (max 1000, min 1, default 1000)")
	fmt.Println("  	--csv Export output to csv")
	fmt.Println("	--json Export output to json")
	fmt.Println("	--cli Export output to cli")
}

func main() {
	bannerOutput()
	argsLen := len(os.Args)
	targets := []string{}
	partition := 0
	exportSettings := journalparser.ExportSettings{
		CSV:  false,
		JSON: false,
		CLI:  false,
	}
	maxBatch := 1000
	for i := 0; i < argsLen; i++ {
		if os.Args[i] == "--help" || os.Args[i] == "-h" {
			printHelpMessage()
			return
		}
		if os.Args[i] == "--target" || os.Args[i] == "-t" {
			if i+1 >= argsLen {
				fmt.Println("Error: Missing argument for --target or -t")
				printHelpMessage()
				return
			}
			if os.Args[i+1] == "" {
				fmt.Println("Error: Missing argument for --target or -t")
				printHelpMessage()
				return
			}
			targets = append(targets, os.Args[i+1])
			i += 1
			continue
		}
		if os.Args[i] == "--max-batch" || os.Args[i] == "-mb" {
			if i+1 >= argsLen {
				fmt.Println("Error: Missing argument for --max-batch or -mb")
				printHelpMessage()
				return
			}
			flagMaxBatch, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				fmt.Println("Error: Invalid argument for --max-batch or -mb")
				printHelpMessage()
				return
			}
			if flagMaxBatch < 1 || flagMaxBatch > 1000 {
				fmt.Println("Error: Invalid argument for --max-batch or -mb")
				printHelpMessage()
				return
			}
			maxBatch = flagMaxBatch
			i += 1
			continue
		}
		if os.Args[i] == "--partition" || os.Args[i] == "-p" {
			if i+1 >= argsLen {
				fmt.Println("Error: Missing argument for --partition or -p")
				printHelpMessage()
				return
			}
			flagPartition, err := strconv.Atoi(os.Args[i+1])
			if err != nil {
				fmt.Println("Error: Invalid argument for --partition or -p")
				printHelpMessage()
				return
			}
			if flagPartition < 0 {
				fmt.Println("Error: Invalid argument for --partition or -p")
				printHelpMessage()
				return
			}
			partition = flagPartition
			i += 1
			continue
		}
		if os.Args[i] == "--csv" {
			exportSettings.CSV = true
			continue
		}
		if os.Args[i] == "--json" {
			exportSettings.JSON = true
			continue
		}
		if os.Args[i] == "--cli" {
			exportSettings.CLI = true
			continue
		}
	}

	if len(targets) == 0 {
		fmt.Println("Error: Missing argument for --target or -t")
		printHelpMessage()
		return
	}
	output := time.Now().Format("output_2006-01-02_15-04-05")
	fmt.Println("Mode")
	fmt.Println("Targets: ", strings.Join(targets, ", "))
	fmt.Println("Partition: ", partition)
	fmt.Println("Output: ", output)
	fmt.Println("")
	fmt.Println("Creating Database...")

	repository := journalparser.NewJournalRepository(maxBatch)

	err := os.Mkdir("./"+output, 0755)
	if err != nil {
		fmt.Println("Failed to create output directory: ", err)
		return
	}

	err = repository.ConnectToDB(output)
	if err != nil {
		fmt.Println("Error connecting to database: ", err)
		return
	}
	defer repository.Close()
	fmt.Println("Database created")
	fmt.Println("Creating SQL Tables...")
	err = repository.CreateTables()
	if err != nil {
		fmt.Println("Error creating SQL tables: ", err)
		return
	}
	fmt.Println("SQL Tables created")
	exporter := journalparser.NewExporter(exportSettings.CSV, exportSettings.JSON, exportSettings.CLI, output)
	fmt.Println("Starting parsing...")
	jp := journalparser.NewJournalParser(targets, partition, output, exporter, repository)
	jp.Parse()
}
