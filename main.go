package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/yodeman/termdict/tui"
)

// message shown when dictionary database files are missing in the
// expected application directory.
const dbaseCheckMsg = `
Words database not found in the expected directory.

Would you like to download the words database [Y|y]es or [N|n]o: `

// Total numbers of the expected dictionary database files.
const dbaseLen = 26

// Program entry point.
func main() {
	tui.DictDbase, tui.DictWords = setup()

	tui.RenderLayout()
}

// setup checks if the words database exists on the user's system.
// It attempts to download the words database based on the input
// received from the user, provided there is internet connection.
func setup() (map[string]tui.DictEntity, []string) {
	_, err := os.Stat(tui.DbaseDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(tui.DbaseDir, 0774)
		if err != nil {
			log.Fatalf("Error creating directory %s.\n%v\n", tui.DbaseDir, err)
		}
	}

	jsonFiles, err := fs.Glob(os.DirFS(tui.DbaseDir), "wb1913_*.json")
	if err != nil {
		log.Fatalf("Error accessing directory %q.\n%v\n", tui.DbaseDir, err)
	}

	if len(jsonFiles) < dbaseLen {
		var prompt string
		fmt.Fprintf(os.Stdout, dbaseCheckMsg)
		fmt.Scanf("%s", &prompt)

		if strings.HasPrefix(strings.ToLower(prompt), "n") {
			fmt.Println("Exiting...\n")
			os.Exit(0)
		}

		// Download words database
		fmt.Println("Downloading words database...")
		err = tui.FetchDbase()
		if err != nil {
			log.Fatalf("%v", err)
		}
	}

	jsonFiles, err = fs.Glob(os.DirFS(tui.DbaseDir), "wb1913_*.json")
	if err != nil {
		log.Fatalf("Error accessing directory %q.\n%v\n", tui.DbaseDir, err)
	}

	words := []string{}
	dbase := map[string]tui.DictEntity{}
	words = loadWords(tui.DbaseDir, jsonFiles, dbase, words)

	return dbase, words
}

// loadWords loads the dictionary words database
func loadWords(
	rootPath string,
	wordsDbase []string,
	dbase map[string]tui.DictEntity,
	words []string,
) []string {
	for _, f := range wordsDbase {
		openedFile, err := os.Open(rootPath + f)
		words := map[string]tui.DictEntity{}
		if err != nil {
			log.Printf("Error loading %s.\n%v\n", rootPath+f, err)
			continue
		}

		if err = json.NewDecoder(openedFile).Decode(&words); err != nil {
			log.Printf("Error decoding json file %s.\n%v\n", rootPath+f, err)
			continue
		}

		maps.Copy(dbase, words)
	}

	for k, _ := range dbase {
		words = append(words, strings.ToLower(k))
	}
	slices.Sort(words)
	return words
}
