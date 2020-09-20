package utility

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func FetchContent(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if err = response.Body.Close(); err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

func DownloadFile(from string, to string) error {
	metaInformationPattern := regexp.MustCompile(`^https?://.*/(.*\.[^?]*)\??.*$`)
	if !metaInformationPattern.MatchString(from) {
		return errors.New("invalid url")
	}
	metaInformation := metaInformationPattern.FindStringSubmatch(from)
	filename := metaInformation[1]
	response, err := http.Get(from)
	if err != nil {
		return err
	}
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(to+filename, bodyBytes, 0644); err != nil {
		return nil
	}
	if err = response.Body.Close(); err != nil {
		return err
	}
	return nil
}

func removeHLSSuffix(indexFile string) error {
	pattern := regexp.MustCompile(`^(.*.ts)\?.*$`)
	content, e := ioutil.ReadFile(indexFile)
	if e != nil {
		return e
	}
	lines := strings.Split(string(content), "\n")

	var builder strings.Builder
	for i := range lines {
		if pattern.MatchString(lines[i]) {
			if _, e := builder.WriteString(pattern.FindStringSubmatch(lines[i])[1] + "\n"); e != nil {
				return e
			}
		} else {
			if _, e := builder.WriteString(lines[i] + "\n"); e != nil {
				return e
			}
		}
	}
	if e := ioutil.WriteFile(indexFile, []byte(builder.String()), 0644); e != nil {
		return e
	}
	return nil
}

func CompareHLS(indexFile string, directory string) error {
	fileBytes, e := ioutil.ReadFile(indexFile)
	if e != nil {
		return e
	}
	lines := strings.Split(string(fileBytes), "\n")
	for i := range lines {
		if strings.HasSuffix(lines[i], ".ts") {
			if !FileExist(directory + lines[i]) {
				fmt.Println(lines[i])
			}
		}
	}
	return nil
}

func downloadHLSIndex(url, destDir string) ([]string, error) {
	metaInformationPattern := regexp.MustCompile(`^(https?://.*/)(.*\.m3u8)\??(.*)$`)
	tlsFilePattern := regexp.MustCompile(`^(.*\.ts)\??(.*)$`)
	cryptoPattern := regexp.MustCompile(`"(.*\.ts)"`)
	indexPattern := regexp.MustCompile(`^(.*\.m3u8)\??(.*)$`)

	metaInformation := metaInformationPattern.FindStringSubmatch(url)
	path := metaInformation[1]
	filename := metaInformation[2]

	result := make([]string, 0)
	if err := DownloadFile(url, destDir); err != nil {
		return nil, err
	}
	content, err := ioutil.ReadFile(destDir + filename)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	for i := range lines {
		line := strings.TrimSpace(lines[i])
		if cryptoPattern.MatchString(line) {
			result = append(result, cryptoPattern.FindStringSubmatch(line)[1])
		} else if tlsFilePattern.MatchString(line) {
			result = append(result, line)
		} else if indexPattern.MatchString(line) {
			matches := indexPattern.FindStringSubmatch(line)
			partition, err := downloadHLSIndex(path+matches[1]+"?"+matches[2], destDir)
			if err != nil {
				return nil, err
			}
			for j := range partition {
				result = append(result, partition[j])
			}
		}
	}
	if e := removeHLSSuffix(destDir + filename); e != nil {
		return result, e
	}
	return result, nil
}

//func mergeHLS(indexFile string) error {
//	if err := os.Chdir(filepath.Dir(indexFile)); err != nil {
//		return err
//	}
//	cmd := exec.Command("ffmpeg", "-i", indexFile, "-c", "copy", "result.mp4")
//	if err := cmd.Run(); err != nil {
//		return err
//	}
//	if err := os.Chdir(filepath.Dir(os.Args[0])); err != nil {
//		return err
//	}
//	return nil
//}

func DownloadHLS(url string, destDir string) error {
	if e := os.Mkdir(destDir+"_go_temp", os.ModePerm); e != nil {
		return e
	}
	pattern := regexp.MustCompile(`^(https?://.*/)(.*\.m3u8)\??(.*)$`)
	meta := pattern.FindStringSubmatch(url)
	chunkFiles, e := downloadHLSIndex(url, destDir+"_go_temp/")
	if e != nil {
		return e
	}

	for i := range chunkFiles {
		fmt.Printf("Downloading: %s %d/%d\n", chunkFiles[i], i, len(chunkFiles)-1)
		for j := 0; j < 3; j++ {
			if j > 0 {
				fmt.Printf("Retry Downloading: %s %d/%d\n", chunkFiles[i], i, len(chunkFiles)-1)
			}
			err := DownloadFile(meta[1]+chunkFiles[i], destDir+"_go_temp/")
			if err == nil {
				break
			}
		}
	}
	//if e := mergeHLS(destDir + "_go_temp/" + meta[2]); e != nil {
	//	return e
	//}
	//source, e := os.Open(destDir + "_go_temp/result.mp4")
	//if e != nil {
	//	return e
	//}
	//destination, e := os.Create(destDir + "result.mp4")
	//if e != nil {
	//	return e
	//}
	//if _, e := io.CopyBuffer(destination, source, make([]byte, 1024)); e != nil {
	//	return e
	//}
	//if e := source.Close(); e != nil {
	//	return e
	//}
	//if e := destination.Close(); e != nil {
	//	return e
	//}
	//if e := os.RemoveAll(destDir + "_go_temp/"); e != nil {
	//	return e
	//}
	return nil
}

func DownloadDocument(url string) error {
	pageContent, e := http.Get(url)
	if e != nil {
		return e
	}

	pageContentBytes, e := ioutil.ReadAll(pageContent.Body)
	if e != nil {
		return e
	}
	if e := pageContent.Body.Close(); e != nil {
		return e
	}

	pageContentBytes = bytes.TrimLeft(pageContentBytes, "wenku_1(")
	pageContentBytes = bytes.TrimRight(pageContentBytes, ")")
	page := make(map[string]interface{})
	if e := json.Unmarshal(pageContentBytes, &page); e != nil {
		return e
	}
	pageBody := page["body"].([]interface{})
	for i := range pageBody {
		fmt.Print(pageBody[i].(map[string]interface{})["c"])
	}
	return nil
}

func DownloadByProxy(from, to string) error {
	pattern := regexp.MustCompile(`^https?://.*/(.*\.[^?]*)\??.*$`)
	filename := pattern.FindStringSubmatch(from)[1]

	proxyURL, e := url.Parse("socks5://127.0.0.1:1080")
	if e != nil {
		return e
	}
	http.DefaultTransport = &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	request, e := http.NewRequest("GET", from, nil)
	if e != nil {
		return e
	}
	request.Header.Set("Referer", "https://avbebe.com")
	response, e := http.DefaultClient.Do(request)
	file, e := os.Create(to + "/" + filename)
	if e != nil {
		return e
	}
	if _, e := io.Copy(file, response.Body); e != nil && e != io.EOF {
		return e
	}
	if e := file.Close(); e != nil {
		return e
	}
	if e := response.Body.Close(); e != nil {
		return e
	}
	return nil
}

func DownloadComic(imageUrl, destination string, begin, end int, proxy bool) error {
	client := &http.Client{}
	if proxy {
		proxyURL, e := url.Parse("socks5://127.0.0.1:1080")
		if e != nil {
			return e
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}
	request, e := http.NewRequest("GET", imageUrl, nil)
	if e != nil {
		return e
	}
	response, e := client.Do(request)
	if e != nil {
		return e
	}
	body, e := ioutil.ReadAll(response.Body)
	if e != nil {
		return nil
	}
	if e := response.Body.Close(); e != nil {
		return e
	}

	pattern := regexp.MustCompile(`https://mi\.404cdn\.com/.*/(.*\.(?:jpg)|(?:png))`)
	images := pattern.FindAllSubmatch(body, -1)

	tempDir := filepath.Join(destination, "_go_temp")

	if e := os.Mkdir(tempDir, os.ModePerm); e != nil {
		if !os.IsExist(e) {
			return e
		}
	}

	images = images[begin:end]
	for i := range images {
		fmt.Println("Downloading: " + string(images[i][0]))
		getImage, e := http.NewRequest("GET", string(images[i][0]), nil)
		if e != nil {
			break
		}
		getResponse, e := client.Do(getImage)
		if e != nil {
			break
		}
		image, e := ioutil.ReadAll(getResponse.Body)
		path := filepath.Join(tempDir, string(images[i][1]))
		if e := ioutil.WriteFile(path, image, os.ModePerm); e != nil {
			return e
		}

		if e := getResponse.Body.Close(); e != nil {
			return e
		}
	}
	return nil
}
