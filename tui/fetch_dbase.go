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

var remoteFiles = []string{
   "wb1913_a.json", "wb1913_b.json", "wb1913_c.json", "wb1913_d.json",
   "wb1913_e.json", "wb1913_f.json", "wb1913_g.json", "wb1913_h.json",
   "wb1913_i.json", "wb1913_j.json", "wb1913_k.json", "wb1913_l.json",
   "wb1913_m.json", "wb1913_n.json", "wb1913_o.json", "wb1913_p.json",
   "wb1913_q.json", "wb1913_r.json", "wb1913_s.json", "wb1913_t.json",
   "wb1913_u.json", "wb1913_v.json", "wb1913_w.json", "wb1913_x.json",
   "wb1913_y.json", "wb1913_z.json"}

const (
    remoteURL = "https://raw.githubusercontent.com/yodeman/termdict/main/word_dbase/json/"
    // file to track database that has changed, to determine the files to be
    // downloaded.
    dbaseTracker = "https://raw.githubusercontent.com/yodeman/termdict/main/word_dbase/changes_tracker.json"
)

func UpdateDbase() (err error) {
    response, err := http.Get(dbaseTracker)
    if err != nil {
        return err
    }
    defer response.Body.Close()

    if response.StatusCode == http.StatusOK {
        res := map[string][]string{}
        err := json.NewDecoder(response.Body).Decode(&res)
        if err != nil {
            return fmt.Errorf("Failed to fetch database changes.")
        }
        remoteFiles = res["changes"]
        if len(remoteFiles) == 0 {
            return fmt.Errorf("Local database is up to date.\nPress escape to exit.")
        }
        err = FetchDbase()
    }
    return err
}

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
    } 

    return err
}

func downloadAndSave(ch chan<- string, savePath, url string) {
    const timeout = 10 * time.Second
    deadline := time.Now().Add(timeout)
    for tries := 0; time.Now().Before(deadline); tries++ {
        response, err := http.Get(url)
        if err != nil {
            continue // retry
        }
        defer response.Body.Close()

        if response.StatusCode == http.StatusOK {
            result := map[string]DictEntity{}
            err := json.NewDecoder(response.Body).Decode(&result)
            if err != nil {
                continue // retry
            }
            encoding, err := json.MarshalIndent(result, "", "    ")
            if err != nil {
                continue // retry
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
            }
            ch<- fmt.Sprint()
            return
        }
    }
    ch<- fmt.Sprintf("Error getting %s after %s.\n\n", url, timeout)
}
