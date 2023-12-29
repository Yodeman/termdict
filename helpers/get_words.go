// Download and save words from The Online Plain Text English Dictionary
// website

package main

import (
	"fmt"
	"io"
	//    "log"
	"net/http"
	"os"
)

func downloadAndSave(url string, savePath string, ch chan<- string) {
	response, err := http.Get(url)
	if err != nil {
		ch <- fmt.Sprintf("Error fetching from %s.\nError:%v\n", url, err)
		return
	}

	if response.StatusCode != http.StatusOK {
		ch <- fmt.Sprintf("Error fetching from %s.\nError:%v\n",
			url, response.StatusCode)
		return
	}

	saveFile, err := os.OpenFile(savePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC,
		0666)
	if err != nil {
		ch <- fmt.Sprintf("Error opening %q\n", save_path)
		return
	}

	writtenBytes, err := io.Copy(saveFile, response.Body)
	response.Body.Close()
	if err != nil {
		ch <- fmt.Sprintf("Error writing to: %q\n", savePath)
		return
	}

	ch <- fmt.Sprintf("Wrote %d bytes to %q\n", writtenBytes, savePath)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage:\n\tget_words SAVE_DIR\n")
		os.Exit(1)
	}
	savePath := os.Args[1]
	urlPrefix := "https://www.mso.anu.edu.au/~ralph/OPTED/v003/"
	resourceLocations := []string{
		"wb1913_a.html", "wb1913_b.html", "wb1913_c.html", "wb1913_d.html",
		"wb1913_e.html", "wb1913_f.html", "wb1913_g.html", "wb1913_h.html",
		"wb1913_i.html", "wb1913_j.html", "wb1913_k.html", "wb1913_l.html",
		"wb1913_m.html", "wb1913_n.html", "wb1913_o.html", "wb1913_p.html",
		"wb1913_q.html", "wb1913_r.html", "wb1913_s.html", "wb1913_t.html",
		"wb1913_u.html", "wb1913_v.html", "wb1913_w.html", "wb1913_x.html",
		"wb1913_y.html", "wb1913_z.html"}

	ch := make(chan string)

	for _, r := range resourceLocations {
		go downloadAndSave(urlPrefix+r, savePath+r, ch)
	}

	for range resourceLocations {
		fmt.Fprintf(os.Stderr, <-ch)
	}
}
