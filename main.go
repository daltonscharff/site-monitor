package main

/*
Checks to see if a website has changed since last run.
Should be used as a cronjob.

Usage: site-monitor.exe --url http://www.google.com
*/

import (
	"flag"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/viper"
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
	err := ioutil.WriteFile(fileName, lines, 0644)
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

func sendTextAlert(siteUrl string, twilioAccountId string, twilioAuthToken string, fromNumber string, toNumber string) *http.Response {
	twilioUrl := fmt.Sprintf("https://%s:%s@api.twilio.com/2010-04-01/Accounts/%s/Messages.json", twilioAccountId, twilioAuthToken, twilioAccountId)

	resp, err := http.PostForm(twilioUrl, url.Values{
		"From": {fromNumber},
		"To":   {toNumber},
		"Body": {"WEBSITE UPDATE%0a" + siteUrl},
	})
	if err != nil {
		panic(err)
	}

	return resp
}

func main() {
	urlPtr := flag.String("url", "", "FQDN to watch for changes")
	dirPtr := flag.String("dir", "./history", "folder to store site versions")
	flag.Parse()

	url := *urlPtr
	dir := *dirPtr

	err := os.MkdirAll(dir, 0644)
	if err != nil {
		panic(err)
	}

	viper.SetConfigFile("./config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	urlHash := strconv.FormatInt(int64(hash(url)), 10)

	resp := getHttpResponse(url)

	prevVersionNum := previousVersion(urlHash, dir)

	oldFileName := fmt.Sprintf("%s/%s.v%d.html", dir, urlHash, prevVersionNum)
	fileName := fmt.Sprintf("%s/%s.v%d.html", dir, urlHash, prevVersionNum+1)

	if prevVersionNum <= 0 {
		writeFile(fileName, resp)
		fmt.Println("New file")
	} else if compareStringToFile(string(resp), oldFileName) == 0 {
		writeFile(fileName, resp)
		fmt.Println("New version")
		sendTextAlert(url, viper.GetString("twilio.accountId"), viper.GetString("twilio.authToken"), viper.GetString("twilio.phoneNumber"), viper.GetString("phoneNumber"))
	} else {
		fmt.Println("No change")
	}
}
