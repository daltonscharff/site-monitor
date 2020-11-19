package main

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func getHttpResponse(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func previousVersion(urlHash string, dir string) int {
	fileList, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	patternString := fmt.Sprintf(`%s\.v(?P<version>\d+).html`, urlHash)
	pattern := regexp.MustCompile(patternString)

	v := 0
	for _, file := range fileList {
		match := pattern.FindStringSubmatch(file.Name())
		if len(match) > 0 {
			i, _ := strconv.Atoi(match[1])
			if i > v {
				v = i
			}
		}
	}
	return v
}

func writeFile(fileName string, lines []byte) {
	err := ioutil.WriteFile(fileName, lines, 0777)
	if err != nil {
		panic(err)
	}
}

func compareStringToFile(text string, fileName string) int {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	return strings.Compare(text, string(content))
}

func main() {
	// url := "https://time.gov/"
	url := "https://www.google.com"
	versionDir := "./versions"

	urlHash := strconv.FormatInt(int64(hash(url)), 10)

	resp := getHttpResponse(url)

	prevVersionNum := previousVersion(urlHash, versionDir)

	oldFileName := fmt.Sprintf("%s/%s.v%d.html", versionDir, urlHash, prevVersionNum)
	fileName := fmt.Sprintf("%s/%s.v%d.html", versionDir, urlHash, prevVersionNum+1)

	if prevVersionNum <= 0 {
		writeFile(fileName, resp)
		fmt.Println("New file")
	} else if compareStringToFile(string(resp), oldFileName) == 0 {
		writeFile(fileName, resp)
		fmt.Println("New version")
		// send alert
	} else {
		fmt.Println("No change")
	}
}
