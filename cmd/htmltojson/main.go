// Command htmltojson parses the HTML pages downloaded from The Online
// Plain Text English Dictionary (OPTED) and extracts words, parts of
// speech and definitions into the JSON database used by TermDict.
//
// Usage:
//
//	htmltojson HTML_DIR
//
// Each wb1913_<letter>.html in HTML_DIR produces a sibling
// wb1913_<letter>.json.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yodeman/termdict/internal/dict"
)

// Headwords may contain apostrophes, hyphens and periods (e.g. "can't",
// "aide-de-camp", "etc."). Earlier revisions matched \w+ only, silently
// dropping every such entry from the database.
var entryPattern = regexp.MustCompile(
	`(?m)<P><B>(?P<word>[A-Za-z][A-Za-z'’.\-]*)</B>\s+\(<I>(?P<pos>.*)</I>\)\s+(?P<def>.+)</P>\s*$`)

func parseHTML(pattern *regexp.Regexp, filePath string, ch chan<- string) {
	openedFile, err := os.Open(filePath)
	if err != nil {
		ch <- fmt.Sprintf("Error opening %q\n", filePath)
		return
	}

	readBytes, err := io.ReadAll(openedFile)
	closeErr := openedFile.Close()
	if err != nil {
		ch <- fmt.Sprintf("Error reading %q\n", filePath)
		return
	}
	if closeErr != nil {
		ch <- fmt.Sprintf("Error closing %q\n", filePath)
		return
	}

	wordIdx := pattern.SubexpIndex("word")
	posIdx := pattern.SubexpIndex("pos")
	defIdx := pattern.SubexpIndex("def")

	dictEntities := map[string]dict.Entity{}
	for _, submatches := range pattern.FindAllSubmatchIndex(readBytes, -1) {
		word := strings.ToLower(strings.Trim(
			string(readBytes[submatches[wordIdx*2]:submatches[wordIdx*2+1]]), "-"))
		pos := string(readBytes[submatches[posIdx*2]:submatches[posIdx*2+1]])
		def := string(readBytes[submatches[defIdx*2]:submatches[defIdx*2+1]])

		entity, ok := dictEntities[word]
		if !ok {
			entity = dict.Entity{Word: word}
		}
		entity.WordDefinitions = append(entity.WordDefinitions, dict.Definition{
			PartOfSpeech:   pos,
			WordDefinition: def,
		})
		dictEntities[word] = entity
	}

	encoding, err := json.MarshalIndent(dictEntities, "", "    ")
	if err != nil {
		ch <- fmt.Sprintf("Error occurred while encoding file %q to json: %v\n",
			filePath, err)
		return
	}

	outFile := strings.TrimSuffix(filePath, ".html") + ".json"
	saveFile, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		ch <- fmt.Sprintf("Error opening output file: %q\n", outFile)
		return
	}

	writtenBytes, err := saveFile.Write(encoding)
	closeErr = saveFile.Close()
	if err != nil {
		ch <- fmt.Sprintf("Error writing to output file %q.\n", outFile)
		return
	}
	if closeErr != nil {
		ch <- fmt.Sprintf("Error closing output file %q.\n", outFile)
		return
	}

	ch <- fmt.Sprintf("Wrote %d bytes to %q\n", writtenBytes, outFile)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage:\n\thtmltojson HTML_DIR\n")
		os.Exit(1)
	}

	filesDir := os.DirFS(os.Args[1])
	htmlFiles, err := fs.Glob(filesDir, "wb1913_*.html")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching directory %q\n", os.Args[1])
		os.Exit(1)
	}
	if len(htmlFiles) == 0 {
		fmt.Fprintf(os.Stderr, "No wb1913_*.html files found in %q\n", os.Args[1])
		os.Exit(1)
	}

	ch := make(chan string, len(htmlFiles))
	for _, f := range htmlFiles {
		go parseHTML(entryPattern, filepath.Join(os.Args[1], f), ch)
	}

	failed := 0
	for range htmlFiles {
		msg := <-ch
		fmt.Fprint(os.Stderr, msg)
		if msg[0] == 'E' {
			failed++
		}
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "%d/%d files failed to convert.\n", failed, len(htmlFiles))
		os.Exit(1)
	}
}
