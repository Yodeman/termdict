// Command getwords downloads the OPTED HTML pages (Webster's 1913,
// public domain) that cmd/htmltojson converts into the TermDict words
// database.
//
// Usage:
//
//	getwords SAVE_DIR
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const urlPrefix = "https://www.mso.anu.edu.au/~ralph/OPTED/v003/"

var resourceLocations = []string{
	"wb1913_a.html", "wb1913_b.html", "wb1913_c.html", "wb1913_d.html",
	"wb1913_e.html", "wb1913_f.html", "wb1913_g.html", "wb1913_h.html",
	"wb1913_i.html", "wb1913_j.html", "wb1913_k.html", "wb1913_l.html",
	"wb1913_m.html", "wb1913_n.html", "wb1913_o.html", "wb1913_p.html",
	"wb1913_q.html", "wb1913_r.html", "wb1913_s.html", "wb1913_t.html",
	"wb1913_u.html", "wb1913_v.html", "wb1913_w.html", "wb1913_x.html",
	"wb1913_y.html", "wb1913_z.html"}

func downloadAndSave(client *http.Client, url, savePath string, ch chan<- string) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		ch <- fmt.Sprintf("Error building request for %s.\nError:%v\n", url, err)
		return
	}

	response, err := client.Do(request)
	if err != nil {
		ch <- fmt.Sprintf("Error fetching from %s.\nError:%v\n", url, err)
		return
	}
	defer response.Body.Close() //nolint:errcheck // read-only body

	if response.StatusCode != http.StatusOK {
		ch <- fmt.Sprintf("Error fetching from %s.\nError:%v\n",
			url, response.StatusCode)
		return
	}

	saveFile, err := os.OpenFile(savePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		ch <- fmt.Sprintf("Error opening %q\n", savePath)
		return
	}

	writtenBytes, err := io.Copy(saveFile, response.Body)
	closeErr := saveFile.Close()
	if err != nil {
		ch <- fmt.Sprintf("Error writing to: %q\n", savePath)
		return
	}
	if closeErr != nil {
		ch <- fmt.Sprintf("Error closing %q\n", savePath)
		return
	}

	ch <- fmt.Sprintf("Wrote %d bytes to %q\n", writtenBytes, savePath)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage:\n\tgetwords SAVE_DIR\n")
		os.Exit(1)
	}
	savePath := os.Args[1]
	if err := os.MkdirAll(savePath, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory %q:\n%v\n", savePath, err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	ch := make(chan string, len(resourceLocations))

	for _, r := range resourceLocations {
		go downloadAndSave(client, urlPrefix+r, filepath.Join(savePath, r), ch)
	}

	failed := 0
	for range resourceLocations {
		msg := <-ch
		fmt.Fprint(os.Stderr, msg)
		if msg[0] == 'E' {
			failed++
		}
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "%d/%d downloads failed.\n", failed, len(resourceLocations))
		os.Exit(1)
	}
}
