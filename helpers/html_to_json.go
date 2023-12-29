// Parse HTML files downloaded from The Online Plain Text Dictionary
// to extract words, part of speech and definitions, to build a json
// database.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

type Definition struct {
	PartOfSpeech   string `json:"part_of_speech"`
	WordDefinition string `json:"definition"`
}

type DictEntity struct {
	Word            string       `json:"word"`
	Spellings       []string     `json:"alternate_spellings,omitempty"`
	WordDefinitions []Definition `json:"definitions"`
}

func parseHTML(filePath string, ch chan<- string) {
	openedFile, err := os.Open(filePath)
	if err != nil {
		ch <- fmt.Sprintf("Error opening %q\n", filePath)
		return
	}

	readBytes, err := ioutil.ReadAll(openedFile)
	if err != nil {
		ch <- fmt.Sprintf("Error reading %q\n", filePath)
		return
	}

	pattern := regexp.MustCompile(
		`(?m)<P><B>(?P<word>\w+)</B>\s+\(<I>(?P<pos>.*)</I>\)\s+(?P<def>.+)</P>$`)

	template := []byte("$word|$pos|$def")
	dictEntities := map[string]DictEntity{}
	for _, submatches := range pattern.FindAllSubmatchIndex(readBytes, -1) {
		b := []byte{}
		b = pattern.Expand(b, template, readBytes, submatches)
		splittedBytes := bytes.Split(b, []byte("|"))

		word := strings.ToLower(string(splittedBytes[0]))
		pos := string(splittedBytes[1])
		def := string(splittedBytes[2])
		if entity, ok := dictEntities[word]; ok {
			entity.WordDefinitions = append(
				entity.WordDefinitions,
				Definition{
					PartOfSpeech:   pos,
					WordDefinition: def,
				},
			)
			dictEntities[word] = entity
		} else {
			dictEntities[word] = DictEntity{
				Word: word,
				WordDefinitions: []Definition{
					Definition{
						PartOfSpeech:   pos,
						WordDefinition: def,
					},
				},
			}
		}
	}
	encoding, err := json.MarshalIndent(dict_entities, "", "    ")
	if err != nil {
		ch <- fmt.Sprintf(
			"Error occured while encoding file: %q to json.\n",
			file_path)
	}

	outFile := strings.TrimSuffix(filePath, ".html") + ".json"
	saveFile, err := os.OpenFile(outFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		ch <- fmt.Sprintf("Error opening output file: %q\n", outFile)
		return
	}

	writtenBytes, err := saveFile.WriteString(fmt.Sprintf("%s\n", encoding))
	if err != nil {
		ch <- fmt.Sprintln("Error writing to output file.")
		return
	}

	ch <- fmt.Sprintf("Wrote %d bytes to %q\n", writtenBytes, outFile)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage:\n\thtml_to_json HTML_DIR\n")
		os.Exit(1)
	}

	filesDir := os.DirFS(os.Args[1])
	htmlFiles, err := fs.Glob(filesDir, "wb1913_*.html")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching directory %q\n", os.Args[1])
		os.Exit(1)
	}
	ch := make(chan string)
	for _, f := range htmlFiles {
		go parseHTML(os.Args[1]+f, ch)
	}

	for range htmlFiles {
		fmt.Fprintf(os.Stderr, <-ch)
	}
}
