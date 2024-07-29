package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	var quietMode bool
	var dryRun bool
	var trim bool
	var numLines int
	var rewrite bool

	flag.BoolVar(&quietMode, "q", false, "quiet mode (no output at all)")
	flag.BoolVar(&dryRun, "d", false, "don't append anything to the file, just print the new lines to stdout")
	flag.BoolVar(&trim, "t", false, "trim leading and trailing whitespace before comparison")
	flag.BoolVar(&rewrite, "r", false, "rewrite existing destination file to remove duplicates and then append unique lines")
	flag.IntVar(&numLines, "ln", -1, "select number of lines (default is -1 for all lines)")
	flag.Parse()

	fn := flag.Arg(0)

	lines := make(map[string]bool)
	var existingLines, newLines []string
	var originalModTime time.Time
	var fileModified bool

	if fn != "" {
		// Check if the file exists and read it if it does
		fileInfo, err := os.Stat(fn)
		if err == nil {
			originalModTime = fileInfo.ModTime()

			r, err := os.Open(fn)
			if err == nil {
				sc := bufio.NewScanner(r)

				for sc.Scan() {
					line := sc.Text()
					if trim {
						line = strings.TrimSpace(line)
					}
					if line == "" {
						continue // Skip blank lines
					}
					if !lines[line] {
						lines[line] = true
						existingLines = append(existingLines, line)
					}
				}
				r.Close()

				if rewrite && !dryRun {
					// Sort the lines before rewriting
					sort.Strings(existingLines)

					// Rewrite the file with unique lines, removing blank lines
					f, err := os.OpenFile(fn, os.O_TRUNC|os.O_WRONLY, 0644)
					if err != nil {
						fmt.Fprintf(os.Stderr, "failed to open file for rewriting: %s\n", err)
						return
					}
					for _, line := range existingLines {
						fmt.Fprintf(f, "%s\n", line)
					}
					f.Close()
					fileModified = true
				}
			}
		} else if os.IsNotExist(err) {
			// Create the file if it does not exist
			f, err := os.OpenFile(fn, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to create file: %s\n", err)
				return
			}
			f.Close()
			// Note: File will be created with empty content, so no need to read existing lines
		} else {
			fmt.Fprintf(os.Stderr, "failed to stat file: %s\n", err)
			return
		}
	}

	// Read the lines from stdin, print and append them if they're new
	sc := bufio.NewScanner(os.Stdin)
	lineCount := 0

	for sc.Scan() {
		line := sc.Text()
		if trim {
			line = strings.TrimSpace(line)
		}
		if line == "" {
			continue // Skip blank lines
		}
		if lines[line] {
			continue
		}

		// Add the line to the map so we don't get any duplicates from stdin
		lines[line] = true
		newLines = append(newLines, line)

		if !quietMode {
			fmt.Println(line)
		}

		lineCount++
		if numLines > 0 && lineCount >= numLines {
			break
		}
	}

	if !dryRun && fn != "" {
		if len(newLines) > 0 {
			// Append new lines to the file if not in dry run mode
			f, err := os.OpenFile(fn, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open file for appending: %s\n", err)
				return
			}
			defer f.Close()

			for _, line := range newLines {
				fmt.Fprintf(f, "%s\n", line)
			}

			fileModified = true
		}

		if fileModified {
			// Update the file's modification time to the original time if any changes were made
			err := os.Chtimes(fn, time.Now(), originalModTime)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to restore file modification time: %s\n", err)
			}
		}
	}
}
