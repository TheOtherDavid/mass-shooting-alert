package alert

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func SendWLEDPulse() {
	//First get the current WLED settings
	currentWled := getWLEDSettings()

	configFile, err := os.Open("config/wled_red_alert_post.json")
	if err != nil {
		return
	}

	byteValue, _ := ioutil.ReadAll(configFile)
	redAlertPulseString := string(byteValue)

	sendWLEDCommand(redAlertPulseString)
	//Wait a number of seconds and return the lights to their prior state.
	time.Sleep(5 * time.Second)
	sendWLEDCommand(currentWled)
}

func sendWLEDCommand(bodyString string) {
	base_url := os.Getenv("WLED_IP")
	url := base_url + "/json/state"

	var jsonprep = []byte(bodyString)

	_, err := http.Post(url, "application/json", bytes.NewBuffer(jsonprep))
	if err != nil {
		fmt.Println("Oh no, error.")
	}
}

func getWLEDSettings() string {
	base_url := os.Getenv("WLED_IP")
	url := base_url + "/json/state"

	response, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}

	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}
	responseString := string(b)

	return responseString
}
