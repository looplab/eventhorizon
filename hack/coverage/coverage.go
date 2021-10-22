package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	flag.Parse()
	args := flag.Args()

	root, err := os.Getwd()
	if err != nil {
		log.Fatal("coverage: Could not get current working directory")
	}

	out := "coverage.out"

	if len(args) == 1 {
		root = args[0]
	} else if len(args) == 2 {
		root = args[0]
		out = args[1]
	} else {
		// nolint:forbidigo
		fmt.Println("Usage: coverage [root] [out]\n\nCollects all .coverprofile files rooted in [root] and concatenantes them into a single file at [out].\n[root] defaults to the current directory, [out] to 'coverage.out'.")
		os.Exit(1)
	}

	outAbs, err := filepath.Abs(out)
	if err != nil {
		log.Fatal("coverage: Could not canonicalize out path:", err)
	}

	coverage := map[string]struct {
		numStmt, count int
	}{}

	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) != ".coverprofile" {
			return err
		}
		if outAbs == path {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		s := bufio.NewScanner(f)
		for s.Scan() {
			l := s.Text()

			if strings.HasPrefix(l, "mode:") {
				continue
			}

			parts := strings.Split(l, " ")
			if len(parts) != 3 {
				log.Fatal("incorrect coverprofile line:", l)
			}

			numStmt, err := strconv.Atoi(parts[1])
			if err != nil {
				log.Fatal("incorrect num stmt in coverprofile line:", l)
			}
			count, err := strconv.Atoi(parts[2])
			if err != nil {
				log.Fatal("incorrect count in coverprofile line:", l)
			}
			// Add or update line.
			if c, ok := coverage[parts[0]]; ok {
				c.count += count
				coverage[parts[0]] = c
			} else {
				coverage[parts[0]] = struct{ numStmt, count int }{numStmt, count}
			}
		}

		if err := s.Err(); err != nil {
			return err
		}

		return err
	}); err != nil {
		log.Fatal("coverage: could not walk directory structure:", err)
	}

	lines := []string{}
	for l, d := range coverage {
		lines = append(lines, fmt.Sprintf("%s %d %d", l, d.numStmt, d.count))
	}

	sort.Strings(lines)

	if err := ioutil.WriteFile(out, []byte(strings.Join(lines, "\n")), 0666); err != nil {
		log.Fatal("coverage: could not write to out:", err)
	}
}
