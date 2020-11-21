package structs

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// FileVersion holds file history metadata
type FileVersion struct {
	URL     string
	Hash    string
	Version int
	Body    string
}

// SetHash hashes the URL to be used as the file name
func (f *FileVersion) SetHash() {
	h := fnv.New32a()
	h.Write([]byte((*f).URL))
	(*f).Hash = strconv.FormatInt(int64(h.Sum32()), 10)
}

// SetMostRecentVersion returns how many versions of a site are stored in history
func (f *FileVersion) SetMostRecentVersion(dir string) {
	fileList, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	patternString := fmt.Sprintf(`%s\.v(?P<version>\d+).html`, (*f).Hash)
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

	(*f).Version = v
}

// GetFileName turns a hash and version number into a valid file name
func (f *FileVersion) GetFileName() string {
	return fmt.Sprintf("%s.v%d.html", f.Hash, f.Version)
}

// Compare compares the body of two version
func (f FileVersion) Compare(b FileVersion) int {
	return strings.Compare(f.Body, b.Body)
}

// ReadBody reads the body from a file to a version if it exists
func (f *FileVersion) ReadBody(dir string) {
	filePath := fmt.Sprintf("%s/%s", dir, f.GetFileName())

	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		(*f).Body = ""
		return
	} else if err != nil {
		panic(err)
	}

	content, _ := ioutil.ReadFile(filePath)
	(*f).Body = string(content)
}

// WriteBody writes the body from a version to a new file
func (f *FileVersion) WriteBody(dir string) {
	filePath := fmt.Sprintf("%s/%s", dir, f.GetFileName())
	err := ioutil.WriteFile(filePath, []byte(f.Body), 0644)
	if err != nil {
		panic(err)
	}
}
