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
    PartOfSpeech    string `json:"part_of_speech"`
    WordDefinition  string  `json:"definition"`
}
type DictEntity struct {
    Word            string          `json:"word"`
    Spellings       []string        `json:"alternate_spellings,omitempty"`
    WordDefinitions []Definition    `json:"definitions"`
}

func parseHTML(file_path string, ch chan<- string) {
    opened_file, err := os.Open(file_path)
    if err != nil {
        ch<- fmt.Sprintf("Error opening %q\n", file_path)
        return
    }

    read_bytes, err := ioutil.ReadAll(opened_file)
    if err != nil {
        ch<- fmt.Sprintf("Error reading %q\n", file_path)
        return
    }

    pattern := regexp.MustCompile(
        `(?m)<P><B>(?P<word>\w+)</B>\s+\(<I>(?P<pos>.*)</I>\)\s+(?P<def>.+)</P>$`)

    template := []byte("$word|$pos|$def")
    dict_entities := map[string]DictEntity{}
    for _, submatches := range pattern.FindAllSubmatchIndex(read_bytes, -1) {
        b := []byte{}
        b = pattern.Expand(b, template, read_bytes, submatches)
        splitted_bytes := bytes.Split(b, []byte("|"))

        word := strings.ToLower(string(splitted_bytes[0]))
        pos := string(splitted_bytes[1])
        def := string(splitted_bytes[2])
        if entity, ok := dict_entities[word]; ok {
            entity.WordDefinitions = append(
                entity.WordDefinitions,
                Definition{
                    PartOfSpeech: pos,
                    WordDefinition: def,
                },
            )
            dict_entities[word] = entity
        } else {
            dict_entities[word] = DictEntity{
                Word: word,
                WordDefinitions: []Definition{
                    Definition{
                        PartOfSpeech: pos,
                        WordDefinition: def,
                    },
                },
            }
        }
    }
    encoding, err := json.MarshalIndent(dict_entities, "", "    ")
    if err != nil {
        ch<- fmt.Sprintf(
            "Error occured while encoding file: %q to json.\n",
            file_path)
    }

    out_file := strings.TrimSuffix(file_path, ".html")+".json"
    save_file, err := os.OpenFile(out_file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
    if err != nil {
        ch<- fmt.Sprintf("Error opening output file: %q\n", out_file)
        return
    }

    written_bytes, err := save_file.WriteString(fmt.Sprintf("%s\n", encoding))
    if err != nil {
        ch<- fmt.Sprintln("Error writing to output file.")
        return
    }

    ch<- fmt.Sprintf("Wrote %d bytes to %q\n", written_bytes, out_file)
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
