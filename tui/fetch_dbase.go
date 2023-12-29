// Utility to fetch and update words database.

package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Dictionary databse remote file names.
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
	// File to track database that has changed, in order to determine the files to be
	// downloaded.
	dbaseTracker = "https://raw.githubusercontent.com/yodeman/termdict/main/word_dbase/changes_tracker.json"
)

// UpdateDbase attempts to try and update user's local dictionary database,
// by first checking for database files that have changed on the remote repository
// and then download those files if any.
func UpdateDbase() (err error) {
	remoteFiles, err = checkChanges()
	if err != nil {
		return err
	} else if len(remoteFiles) == 0 {
		return fmt.Errorf(`
        Local database is up to date.

                        [yellow::b]press escape to exit.
        `)
	}

	err = FetchDbase()

	return err
}

// checkChanges checks the remote repository for changes in the dictionary
// database. It collects the names of files with changes, if there are any.
func checkChanges() (changes []string, err error) {
	changesDir := strings.TrimSuffix(DbaseDir, "json"+string(os.PathSeparator))
	cf, err := os.Open(changesDir + "changes_tracker.json")
	if err != nil {
		return changes, fmt.Errorf("Error obtaining changes.\n%v\n", err)
	}
	localChanges := map[string]string{}
	err = json.NewDecoder(cf).Decode(&localChanges)
	cf.Close()
	if err != nil {
		return changes, fmt.Errorf("Error obtaining changes.\n%v\n", err)
	}

	response, err := http.Get(dbaseTracker)
	if err != nil {
		return changes, fmt.Errorf("Error obtaining changes.\n%v\n", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		remoteChanges := map[string]string{}
		err = json.NewDecoder(response.Body).Decode(&remoteChanges)
		if err != nil {
			return changes, fmt.Errorf("Error obtaining changes.\n%v\n", err)
		}

		for dbase, ver := range remoteChanges {
			if lVer, ok := localChanges[dbase]; !ok || (ver != lVer) {
				changes = append(changes, dbase)
			}
		}

		return changes, nil
	}

	err = fmt.Errorf("Error obtaining changes.\n%v\n", response.StatusCode)

	return changes, err
}

// FetchDbase downloads dictionary database files contained in the **remoteFiles**
// variable. The user's local database files changes tracker is updated to ensure
// that no download is made on database files without any changes.
func FetchDbase() (err error) {
	ch := make(chan string)
	writer := new(strings.Builder)
	for _, file := range remoteFiles {
		go downloadAndSave(ch, DbaseDir+file, remoteURL+file)
	}

	for range remoteFiles {
		writer.WriteString(<-ch)
	}

	// update local changes tracker file.
	response, err := http.Get(dbaseTracker)
	if err != nil {
		writer.WriteString(fmt.Sprintf("Error obtaining changes.\n%v\n", err))
		return fmt.Errorf("%s", writer.String())
	}
	defer response.Body.Close()
	localChanges := map[string]string{}
	err = json.NewDecoder(response.Body).Decode(&localChanges)
	changesDir := strings.TrimSuffix(DbaseDir, "json"+string(os.PathSeparator))
	cf, err := os.OpenFile(
		changesDir+"changes_tracker.json",
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0666,
	)
	if err != nil {
		writer.WriteString(fmt.Sprintf("Error updating local changes.\n%v\n", err))
		return fmt.Errorf("%s", writer.String())
	}
	defer cf.Close()
	encoding, err := json.MarshalIndent(localChanges, "", "    ")
	if err != nil {
		writer.WriteString(fmt.Sprintf("Error updating local changes.\n%v\n", err))
		return fmt.Errorf("%s", writer.String())
	}
	_, err = cf.WriteString(fmt.Sprintf("%s", encoding))
	if err != nil {
		writer.WriteString(fmt.Sprintf("Error updating local changes.\n%v\n", err))
		return fmt.Errorf("%s", writer.String())
	}

	if writer.Len() != 0 {
		err = fmt.Errorf("%s", writer.String())
	}

	return err
}

// downloadAndSave downloads file in the given url and saves it in the file
// names on the savePath argument.
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
			openedFile, err := os.OpenFile(
				savePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
			if err != nil {
				ch <- fmt.Sprintf(
					"Error opening %s for writing, while getting %s.\n\n",
					savePath, url)
				return
			}
			defer openedFile.Close()
			_, err = openedFile.WriteString(fmt.Sprintf("%s", encoding))
			if err != nil {
				ch <- fmt.Sprintf(
					"Error writing to %s, while getting %s.\n\n",
					savePath, url)
			}
			ch <- fmt.Sprint()
			return
		}
	}
	ch <- fmt.Sprintf("Error getting %s after %s.\n\n", url, timeout)
}
