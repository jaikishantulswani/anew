package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
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
	flag.BoolVar(&rewrite, "r", false, "rewrite existing destination file to remove duplicates and blank lines")
	flag.IntVar(&numLines, "ln", -1, "select number of lines (default is -1 for all lines)")
	flag.Parse()

	fn := flag.Arg(0)

	lines := make(map[string]bool)

	if fn != "" {
		// read the whole file into a map if it exists
		r, err := os.Open(fn)
		if err == nil {
			sc := bufio.NewScanner(r)

			for sc.Scan() {
				line := sc.Text()
				if trim {
					line = strings.TrimSpace(line)
				}
				if line == "" {
					continue // skip blank lines
				}
				lines[line] = true
			}
			r.Close()

			if rewrite && !dryRun {
				// Rewrite the file with unique lines, removing blank lines
				f, err := os.OpenFile(fn, os.O_TRUNC|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to open file for rewriting: %s\n", err)
					return
				}
				for line := range lines {
					fmt.Fprintf(f, "%s\n", line)
				}
				f.Close()
			}
		}
	}

	if !dryRun {
		// re-open the file for appending new stuff if not in dry run mode
		f, err := os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open file for writing: %s\n", err)
			return
		}
		defer f.Close()
	}

	// read the lines from stdin, print and append them if they're new
	sc := bufio.NewScanner(os.Stdin)
	lineCount := 0

	for sc.Scan() {
		line := sc.Text()
		if trim {
			line = strings.TrimSpace(line)
		}
		if line == "" {
			continue // skip blank lines
		}
		if lines[line] {
			continue
		}

		// add the line to the map so we don't get any duplicates from stdin
		lines[line] = true

		if !quietMode {
			fmt.Println(line)
		}
		if !dryRun && fn != "" {
			f, err := os.OpenFile(fn, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to open file for writing: %s\n", err)
				return
			}
			fmt.Fprintf(f, "%s\n", line)
			f.Close()
		}

		lineCount++
		if numLines > 0 && lineCount >= numLines {
			break
		}
	}

	if rewrite && !dryRun {
		// After processing stdin, rewrite the file again to ensure it's updated
		f, err := os.OpenFile(fn, os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open file for final rewriting: %s\n", err)
			return
		}
		for line := range lines {
			fmt.Fprintf(f, "%s\n", line)
		}
		f.Close()
	}
}
