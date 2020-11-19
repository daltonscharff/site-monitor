package main

/*
Checks to see if a website has changed since last run.
Should be used as a cronjob.

Usage: site-monitor.exe --url http://www.google.com --phone +01234567890 --email abc@xyz.com
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

func getHtml(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(body)
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

func parseFlags(siteUrl *string, saveDir *string, phone *string, email *string) {
	flag.StringVar(siteUrl, "url", "", "FQDN to watch for changes")
	flag.StringVar(saveDir, "dir", "./history", "folder to store site versions")
	flag.StringVar(phone, "phone", "", "phone number for sending text updates (optional, requires twilio information config.yaml file)")
	flag.StringVar(email, "email", "", "email address for sending email updates (optional)")
	flag.Parse()

	if len(*siteUrl) == 0 {
		fmt.Println("Please provide a URL to watch")
		os.Exit(1)
	}

	if len(*saveDir) == 0 {
		fmt.Println("Please provide a valid directory to store site versions")
		os.Exit(2)
	}

	if len(*phone) == 10 {
		*phone = fmt.Sprintf("+1%s", *phone)
	}

	err := os.MkdirAll(*saveDir, 0644)
	if err != nil {
		fmt.Println("Could not create save directory", err)
	}
}

func readConfig() {
	viper.SetConfigFile("./config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}

type FileVersion struct {
	url     string
	hash    string
	version int
	body    string
}

func (f *FileVersion) setHash() {
	h := fnv.New32a()
	h.Write([]byte((*f).url))
	(*f).hash = strconv.FormatInt(int64(h.Sum32()), 10)
}

func (f *FileVersion) setMostRecentVersion(dir string) {
	fileList, err := ioutil.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	patternString := fmt.Sprintf(`%s\.v(?P<version>\d+).html`, (*f).hash)
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

	(*f).version = v
}

func (f *FileVersion) getFileName() string {
	return fmt.Sprintf("%s.v%d.html", f.hash, f.version)
}

func (a FileVersion) compare(b FileVersion) int {
	return strings.Compare(a.body, b.body)
}

func (f *FileVersion) readBody(dir string) {
	filePath := fmt.Sprintf("%s/%s", dir, f.getFileName())
	content, err := ioutil.ReadFile(filePath)

	if os.IsNotExist(err) {
		(*f).body = ""
		return
	} else if err != nil {
		panic(err)
	}

	(*f).body = string(content)
}

func (f *FileVersion) writeBody(dir string) {
	filePath := fmt.Sprintf("%s/%s", dir, f.getFileName())
	err := ioutil.WriteFile(filePath, []byte(f.body), 0644)
	if err != nil {
		panic(err)
	}
}

func main() {
	var siteUrl, saveDir, phone, email string

	parseFlags(&siteUrl, &saveDir, &phone, &email)

	var previousVersion FileVersion
	previousVersion.url = siteUrl
	previousVersion.setHash()
	previousVersion.setMostRecentVersion(saveDir)
	previousVersion.readBody(saveDir)

	currentVersion := previousVersion
	currentVersion.version++
	currentVersion.body = getHtml(siteUrl)

	if currentVersion.version == 1 {
		fmt.Println("No previous versions")
		currentVersion.writeBody(saveDir)
	} else if currentVersion.compare(previousVersion) != 0 {
		fmt.Println("Change")
		currentVersion.writeBody(saveDir)

		if len(phone) > 0 {
			readConfig()
		}

		if len(email) > 0 {

		}
	} else {
		fmt.Println("No change")
	}

	// resp := getHttpResponse(url)

	// prevVersionNum := previousVersion(urlHash, dir)

	// oldFileName := fmt.Sprintf("%s/%s.v%d.html", dir, urlHash, prevVersionNum)
	// fileName := fmt.Sprintf("%s/%s.v%d.html", dir, urlHash, prevVersionNum+1)

	// if prevVersionNum <= 0 {
	// 	writeFile(fileName, resp)
	// 	fmt.Println("New file")
	// } else if compareStringToFile(string(resp), oldFileName) == 0 {
	// 	writeFile(fileName, resp)
	// 	fmt.Println("New version")
	// 	sendTextAlert(url, viper.GetString("twilio.accountId"), viper.GetString("twilio.authToken"), viper.GetString("twilio.phoneNumber"), viper.GetString("phoneNumber"))
	// } else {
	// 	fmt.Println("No change")
	// }
}
