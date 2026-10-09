package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/galaco/KeyValues/v2"
)

func main() {
	for _, dir := range []string{"vmt", "vmf", "gameinfo"} {
		readDir(dir)
	}
}

// readDir parses every file in dir and logs its top-level keys.
func readDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		doc, err := read(path)
		if err != nil {
			log.Fatal(err)
		}
		for _, child := range doc.Children() {
			log.Printf("%s: %s", path, child.Key())
		}
	}
}

func read(path string) (*keyvalues.KeyValue, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return keyvalues.NewReader(f).Read()
}
