package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/ini.v1"
)

func main() {
	//Find last triggered date.
	//Access local data file?

	lastShootingCity, lastShootingDate, lastTriggeredDate, err := getLastTriggeredData()
	if err != nil {
		println("Error retrieving data from file.")
	}
	//Hit the MST bucket and get the last updated date
	lastUpdatedDate, err := queryS3Bucket()
	if err != nil {
		println("Error retrieving metadata from S3 bucket.")
	}

	var incidents []Incident
	//TODO: Use the last update date from the file
	//lastUpdatedDate := time.Now()

	//If the last updated date is NEWER than the last triggered date, download the file
	//TODO: Pull this out into its own function.
	if lastTriggeredDate.Before(lastUpdatedDate) {
		incidents, err = getIncidents()
		if err != nil {
			println("Error retrieving incidents from S3 bucket.")
		}
		println("Incidents downloaded.")
	} else {
		println("Not downloading incidents, no new update.")
		println("No shootings this time!")
		return
	}

	//Calculations. Use goroutines?
	//Maybe find the total number of dead/wounded, and compare it to a high mark like 50?
	dead, wounded := extractDailyDeadAndWoundedCount(incidents)
	println(strconv.Itoa(dead))
	println(strconv.Itoa(wounded))

	newShooting := isNewShootingToday(incidents, lastShootingCity, lastShootingDate)

	//If result is true, call WLED
	if newShooting {
		println("Oh no, there's a new shooting!")
		//Do some other stuff
		//Ohhh, should we make this a goroutine? That way we don't have to wait on it to update the file
		sendWLEDPulse()
		//Update the data file to have the latest data
		lastTriggeredIncident := incidents[0]
		lastShootingCity = lastTriggeredIncident.City
		lastShootingDate = lastTriggeredIncident.Date
		lastTriggeredDate = time.Now()
		SetLastTriggeredData(lastShootingCity, lastShootingDate, lastTriggeredDate)
	} else {
		println("No shootings this time!")
	}

}

func getLastTriggeredData() (lastShootingCity string, lastShootingDate time.Time, lastTriggeredDate time.Time, err error) {

	cfg, err := ini.Load("config/data.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	// Classic read of values, default section can be represented as empty string
	lastShootingCity = cfg.Section("").Key("last_shooting_city").String()
	lastShootingDateString := cfg.Section("").Key("last_shooting_date").String()
	lastTriggeredDateString := cfg.Section("").Key("last_triggered_date").String()

	layout := "2006-01-02T15:04:05.000Z"
	lastShootingDate, err = time.Parse(layout, lastShootingDateString)
	if err != nil {
		return
	}
	lastTriggeredDate, err = time.Parse(layout, lastTriggeredDateString)
	if err != nil {
		return
	}

	fmt.Println(lastTriggeredDateString)

	return lastShootingCity, lastShootingDate, lastTriggeredDate, nil
}

func SetLastTriggeredData(lastShootingCity string, lastShootingDate time.Time, lastTriggeredDate time.Time) {
	cfg, err := ini.Load("config/data.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}
	layout := "2006-01-02T15:04:05.000Z"
	lastShootingDateString := lastShootingDate.Format(layout)
	lastTriggeredDateString := lastTriggeredDate.Format(layout)

	cfg.Section("").Key("last_shooting_city").SetValue(lastShootingCity)
	cfg.Section("").Key("last_shooting_date").SetValue(lastShootingDateString)
	cfg.Section("").Key("last_triggered_date").SetValue(lastTriggeredDateString)
	cfg.SaveTo("config/data.ini")
}

func queryS3Bucket() (lastModified time.Time, err error) {
	//So they have an S3 bucket, and we should get the file
	bucket := "mass-shooting-tracker-data"
	// TODO: Dynamically construct this
	year := "2022"
	//Target filename: 2022-data.json
	filename := year + "-data.json"

	accessKey := os.Getenv("AWS_ACCESS_KEY")
	secretKey := os.Getenv("AWS_SECRET_KEY")

	if accessKey == "" {
		errors.New("Env variable access_key not found")
		return time.Time{}, err
	}

	if secretKey == "" {
		errors.New("Env variable secret_key not found")
		return time.Time{}, err
	}

	client := s3.New(s3.Options{
		Region:      "us-east-2",
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	})

	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(filename),
	}

	p := s3.NewListObjectsV2Paginator(client, params)
	// Iterate through the Amazon S3 object pages.
	var s3File S3File

	for p.HasMorePages() {
		// next page takes a context
		page, err := p.NextPage(context.TODO())
		if err != nil {
			fmt.Errorf("failed to get a page, %w", err)
		}
		//Take first (probably only) record
		file := page.Contents[0]
		s3File = S3File{
			LastModified: *file.LastModified,
			Key:          *file.Key,
		}
	}

	println(s3File.Key)
	return s3File.LastModified, nil

}

type S3File struct {
	Key          string
	LastModified time.Time
}

func getIncidents() (incidents []Incident, err error) {
	//So they have an S3 bucket, and we should get the file
	bucket := "mass-shooting-tracker-data"
	// TODO: Dynamically construct this
	year := "2022"
	//Target filename: 2022-data.json
	filename := year + "-data.json"

	accessKey := os.Getenv("AWS_ACCESS_KEY")
	secretKey := os.Getenv("AWS_SECRET_KEY")

	client := s3.New(s3.Options{
		Region:      "us-east-2",
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	})

	params := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	}

	result, err := client.GetObject(context.TODO(), params)
	if err != nil {
		return
	}

	defer result.Body.Close()
	body1, err := io.ReadAll(result.Body)
	if err != nil {
		return
	}

	_ = json.Unmarshal([]byte(string(body1)), &incidents)

	incidents, err = convertDateStringToDate(incidents)
	if err != nil {
		return
	}

	return
}

type Incident struct {
	Date       time.Time
	DateString string   `json:"date"`
	Killed     string   `json:"killed"`
	Wounded    string   `json:"wounded"`
	City       string   `json:"city"`
	Names      []string `json:"names"`
	Sources    []string `json:"sources"`
}

func convertDateStringToDate(incidents []Incident) (convertedIncidents []Incident, err error) {
	for _, incident := range incidents {
		layout := "2006-01-02T15:04:05.000Z"

		var incidentDate time.Time
		incidentDate, err = time.Parse(layout, incident.DateString)
		if err != nil {
			return
		}
		incident.Date = incidentDate
		convertedIncidents = append(convertedIncidents, incident)
	}
	return
}

func extractDailyDeadAndWoundedCount(incidents []Incident) (int, int) {
	//TODO: Implement
	return 0, 0
}

func getIncidentsFromToday(incidents []Incident) []Incident {
	var incidentsFromToday []Incident
	t := time.Now()
	currentDate := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	for _, incident := range incidents {
		if incident.Date.Equal(currentDate) {
			incidentsFromToday = append(incidentsFromToday, incident)
		} else {
			//Break out of the loop. We aren't interested in the rest
			return incidentsFromToday
		}
	}
	return incidentsFromToday
}

func isNewShootingToday(incidents []Incident, lastShootingCity string, lastShootingDate time.Time) bool {
	//Determine whether there has been a shooting that meets the criteria
	//Date/City is close enough, since we don't have a real timestamp. Unlikely for multiple shootings in the same city on the same day.
	if len(incidents) == 0 {
		return false
	}
	response := true
	for _, incident := range incidents {
		if incident.City == lastShootingCity && incident.Date == lastShootingDate {
			response = false
		}
	}
	return response
}

func sendWLEDPulse() {
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
