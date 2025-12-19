package main

import (
	"fmt"
	journalparser "journal-parser/pkg/journal-parser"
	"os"
	"strconv"
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
	fmt.Println("  -h, --help    Display this help message")
	fmt.Println("  -t, --target <Directory or filename>  The target to parse")
	fmt.Println("  -p, --partition <Number>  Max strings per file (no partition by default)")
}

func main() {
	bannerOutput()
	argsLen := len(os.Args)
	target := ""
	partition := 0
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
			target = os.Args[i+1]
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
			partition = flagPartition
			i += 1
			continue
		}
	}

	if target == "" {
		fmt.Println("Error: Missing argument for --target or -t")
		printHelpMessage()
		return
	}
	output := time.Now().Format("output_2006-01-02_15-04-05")
	fmt.Println("Mode")
	fmt.Println("Target: ", target)
	fmt.Println("Partition: ", partition)
	fmt.Println("Output: ", output)
	fmt.Println("")
	fmt.Println("Starting parsing...")
	jp := journalparser.NewJournalParser(target, partition, output)
	jp.Parse()
}
