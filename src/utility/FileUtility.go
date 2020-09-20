package utility

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
)

func StartFileServer(path string) error {
	http.Handle("/upload", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, header, _ := r.FormFile("uploadFile")
		fileBytes, _ := ioutil.ReadAll(file)
		_ = ioutil.WriteFile(path+header.Filename, fileBytes, 0644)
		_, _ = fmt.Fprintln(w, `{"result": "success"}`)
		_ = file.Close()
	}))

	http.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := ioutil.ReadAll(r.Body)
		_ = r.Body.Close()
		result := make(map[string]interface{})
		_ = json.Unmarshal(bytes, &result)
		fmt.Println(result)
		response, _ := json.Marshal(result)
		_, _ = fmt.Fprintln(w, string(response))
	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(path))))
	if e := http.ListenAndServe(":8080", nil); e != nil {
		return e
	}
	return nil
}

func FileExist(filename string) bool {
	result := false
	_, err := os.Stat(filename)

	if err == nil {
		result = true
	} else if os.IsNotExist(err) {
		result = false
	}
	return result
}

func BatchRename(path string, pattern *regexp.Regexp) error {
	files, e := ioutil.ReadDir(path)
	if e != nil {
		return e
	}
	for i := range files {
		if pattern.MatchString(files[i].Name()) {
			origin := files[i].Name()
			destination := pattern.FindStringSubmatch(files[i].Name())[1]
			if e := os.Rename(path+origin, path+destination); e != nil {
				return e
			}
		}
	}
	return nil
}
