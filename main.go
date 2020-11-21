package main

/*
Checks to see if a website has changed since last run.
Should be used as a cronjob.

Usage: site-monitor.exe --url http://www.google.com --phone +01234567890 --email abc@xyz.com
*/

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/viper"

	"github.com/daltonscharff/site-monitor/structs"
)

func get(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Cannot reach url")
		os.Exit(3)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(body)
}

func sendText(message string, twilioAccountID string, twilioAuthToken string, fromNumber string, toNumber string) *http.Response {
	twilioURL := fmt.Sprintf("https://%s:%s@api.twilio.com/2010-04-01/Accounts/%s/Messages.json", twilioAccountID, twilioAuthToken, twilioAccountID)

	resp, err := http.PostForm(twilioURL, url.Values{
		"From": {fromNumber},
		"To":   {toNumber},
		"Body": {message},
	})
	if err != nil {
		panic(err)
	}

	return resp
}

func sendEmail(toEmail string, fromEmail string, subject string, message string, sendgridApiKey string) *http.Response {
	sendgridURL := "https://api.sendgrid.com/v3/mail/send"

	data := []byte(`{
		"personalizations": 
		[{"to": [{"email": "` + toEmail + `"}]}],
		"from": {"email": "` + fromEmail + `"},
		"subject": "` + subject + `",
		"content": [{"type": "text/plain", "value": "` + message + `"}]}`)

	req, err := http.NewRequest("POST", sendgridURL, bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+sendgridApiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println(resp)
	return resp
}

func parseFlags(siteURL *string, saveDir *string, phone *string, email *string) {
	flag.StringVar(siteURL, "url", "", "FQDN to watch for changes")
	flag.StringVar(saveDir, "dir", "./history", "folder to store site versions")
	flag.StringVar(phone, "phone", "", "phone number for sending text updates (optional, requires twilio information config.yaml file)")
	flag.StringVar(email, "email", "", "email address for sending email updates (optional)")
	flag.Parse()

	if len(*siteURL) == 0 {
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

func readConfig(phone string, email string) {
	viper.SetConfigFile("./config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	if len(phone) > 0 {
		if !viper.IsSet("twilio.accountId") || !viper.IsSet("twilio.authToken") || !viper.IsSet("twilio.phoneNumber") {
			fmt.Println("Texting a phone requires the following in config.yaml:\ntwilio.accountId\ntwilio.authToken\ntwilio.phoneNumber")
			os.Exit(4)
		}
	}
	if len(email) > 0 {
		if !viper.IsSet("sendgrid.apiKey") || !viper.IsSet("sendgrid.fromEmail") {
			fmt.Println("Sending an email requires the following in config.yaml:\nsendgrid.apiKey\nsendgrid.fromEmail")
			os.Exit(4)
		}
	}
}

func main() {
	var siteURL, saveDir, phone, email string

	parseFlags(&siteURL, &saveDir, &phone, &email)
	readConfig(phone, email)

	var previousVersion structs.FileVersion
	previousVersion.URL = siteURL
	previousVersion.SetHash()
	previousVersion.SetMostRecentVersion(saveDir)
	previousVersion.ReadBody(saveDir)

	currentVersion := previousVersion
	currentVersion.Version++
	currentVersion.Body = get(siteURL)

	if currentVersion.Version == 1 {
		fmt.Println("No previous versions")
		currentVersion.WriteBody(saveDir)
	} else if currentVersion.Compare(previousVersion) != 0 {
		fmt.Println("Change")
		currentVersion.WriteBody(saveDir)

		if len(phone) > 0 {
			message := "WEBSITE UPDATE\n" + siteURL
			sendText(message, viper.GetString("twilio.accountId"), viper.GetString("twilio.authToken"), viper.GetString("twilio.phoneNumber"), phone)
		}
		if len(email) > 0 {
			sendEmail(email, viper.GetString("sendgrid.fromEmail"), "WEBSITE UPDATE", siteURL, viper.GetString("sendgrid.apiKey"))
		}
	} else {
		fmt.Println("No change")
	}
}
