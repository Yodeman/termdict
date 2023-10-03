// Utility to fetch words database

package tui

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"
    "strings"
)

var remoteFiles = [26]string{
   "wb1913_a.json", "wb1913_b.json", "wb1913_c.json", "wb1913_d.json",
   "wb1913_e.json", "wb1913_f.json", "wb1913_g.json", "wb1913_h.json",
   "wb1913_i.json", "wb1913_j.json", "wb1913_k.json", "wb1913_l.json",
   "wb1913_m.json", "wb1913_n.json", "wb1913_o.json", "wb1913_p.json",
   "wb1913_q.json", "wb1913_r.json", "wb1913_s.json", "wb1913_t.json",
   "wb1913_u.json", "wb1913_v.json", "wb1913_w.json", "wb1913_x.json",
   "wb1913_y.json", "wb1913_z.json"}

const remoteURL = "https://raw.githubusercontent.com/yodeman/termdict/main/word_dbase/json/"

func FetchDbase() (err error) {
    ch := make(chan string)
    writer := new(strings.Builder)
    for _, file := range remoteFiles {
        go downloadAndSave(ch, DbaseDir+file, remoteURL+file)
    }

    for range remoteFiles {
        writer.WriteString(<-ch)
    }

    if writer.Len() != 0 {
        err = fmt.Errorf("%s", writer.String())
    } else {
        err = nil
    }

    return err
}

func downloadAndSave(ch chan<- string, savePath, url string) {
    const timeout = 10 * time.Second
    deadline := time.Now().Add(timeout)
    for tries := 0; time.Now().Before(deadline); tries++ {
        response, err := http.Get(url)
        defer response.Body.Close()

        if (err == nil) && (response.StatusCode == http.StatusOK) {
            result := map[string]DictEntity{}
            err := json.NewDecoder(response.Body).Decode(&result)
            if err != nil {
                continue
            }
            encoding, err := json.MarshalIndent(result, "", "    ")
            if err != nil {
                continue
            }
            openedFile, err := os.OpenFile(savePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
            if err != nil {
                ch<- fmt.Sprintf(
                    "Error opening %s for writing, while getting %s.\n\n",
                    savePath, url)
                return
            }
            _, err = openedFile.WriteString(fmt.Sprintf("%s", encoding))
            if err != nil {
                ch<- fmt.Sprintf(
                    "Error writing to %s, while getting %s.\n\n",
                    savePath, url)
                return
            }
        }
    }
    ch<- fmt.Sprintf("Error getting %s after %s.\n\n", url, timeout)
}
