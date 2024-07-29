package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"bytes"
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

	var originalContent bytes.Buffer

	// Check if file exists
	fileExists := false
	if fn != "" {
		if _, err := os.Stat(fn); err == nil {
			fileExists = true
		}
	}

	if fileExists {
		// Read the whole file into a map if it exists
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
					originalContent.WriteString(line + "\n")
				}
			}
			r.Close()

			if rewrite && !dryRun {
				// Create a temporary file to avoid modifying the original file's timestamp
				tmpFileName := fn + ".tmp"
				f, err := os.OpenFile(tmpFileName, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to open temporary file for rewriting: %s\n", err)
					return
				}

				for _, line := range existingLines {
					fmt.Fprintf(f, "%s\n", line)
				}
				f.Close()

				// Read the new content and compare
				var newContent bytes.Buffer
				for _, line := range existingLines {
					newContent.WriteString(line + "\n")
				}
				for _, line := range newLines {
					newContent.WriteString(line + "\n")
				}

				if !bytes.Equal(originalContent.Bytes(), newContent.Bytes()) {
					// Replace the original file with the temporary file if there are changes
					err = os.Rename(tmpFileName, fn)
					if err != nil {
						fmt.Fprintf(os.Stderr, "failed to replace the original file: %s\n", err)
						return
					}
				} else {
					// Clean up the temporary file if no changes
					os.Remove(tmpFileName)
				}
			}
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
		// Open file for appending new lines, create it if it doesn't exist
		f, err := os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open file for appending: %s\n", err)
			return
		}
		defer f.Close()

		for _, line := range newLines {
			fmt.Fprintf(f, "%s\n", line)
		}
	}
}
