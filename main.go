package main

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/tidwall/gjson"
	//"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func main() {
	//Find last triggered date.
	//Access local data file?

	//lastTriggeredDate := time.Now()
	//lastTriggeredCity := "New York"
	lastTriggeredCity, lastTriggeredDate, err := getLastTriggeredData()
	if err != nil {
		println("Oh no, error.")
	}
	//Hit the MST bucket and get the last updated date
	lastUpdatedDate := queryS3Bucket()

	var incidents []Incident
	//TODO: Use the last update date from the file
	//lastUpdatedDate := time.Now()

	//If the last updated date is NEWER than the last triggered date, download the file
	//TODO: Pull this out into its own function.
	if lastTriggeredDate.Before(lastUpdatedDate) {
		incidents, err = getIncidents()
		if err != nil {
			println("Oh no, error.")
		}
	}

	//Calculations. Use goroutines?
	//Maybe find the total number of dead/wounded, and compare it to a high mark like 50?
	dead, wounded := extractDailyDeadAndWoundedCount(incidents)
	println(strconv.Itoa(dead))
	println(strconv.Itoa(wounded))
	incidentsFromToday := getIncidentsFromToday(incidents)
	newShooting := isNewShootingToday(incidentsFromToday, lastTriggeredCity, lastTriggeredDate)

	//If result is true, call WLED
	if newShooting {
		println("Oh no, there's a new shooting!")
		//Do some other stuff
		sendWLEDPulse()
		//Update the data file to have the latest data
		lastTriggeredIncident := incidents[0]
		lastTriggeredCity = lastTriggeredIncident.City
		lastTriggeredDate = lastTriggeredIncident.Date
		SetLastTriggeredData(lastTriggeredCity, lastTriggeredDate)
	} else {
		println("No shootings this time!")
	}
}

func getLastTriggeredData() (lastTriggeredCity string, lastTriggeredDate time.Time, err error) {

	configFile, err := os.Open("config/data.json")
	if err != nil {
		return "", time.Time{}, err
	}

	byteValue, _ := ioutil.ReadAll(configFile)
	lastTriggeredCity = gjson.GetBytes(byteValue, "last_triggered_city").Str
	lastTriggeredDateString := gjson.GetBytes(byteValue, "last_triggered_date").Str

	layout := "2006-01-02T15:04:05.000Z"
	lastTriggeredDate, err = time.Parse(layout, lastTriggeredDateString)
	if err != nil {
		return
	}

	fmt.Println(lastTriggeredDateString)

	return lastTriggeredCity, lastTriggeredDate, nil
}

func SetLastTriggeredData(lastTriggeredCity string, lastTriggeredDate time.Time) {

}

func queryS3Bucket() time.Time {
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
	return s3File.LastModified

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

	println(string(body1))
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

func isNewShootingToday(incidents []Incident, lastTriggeredCity string, lastTriggeredDate time.Time) bool {
	//Determine whether there has been a shooting that meets the criteria
	//Date/City is close enough, since we don't have a real timestamp. Unlikely for multiple shootings in the same day.
	response := true
	for _, incident := range incidents {
		if incident.City == lastTriggeredCity && incident.Date == lastTriggeredDate {
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

	response, err := http.Post(url, "application/json", bytes.NewBuffer(jsonprep))
	if err != nil {
		fmt.Println("Oh no, error.")
	}
	println(response)
}

func getWLEDSettings() string {
	base_url := os.Getenv("WLED_IP")
	url := base_url + "/json/state"

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Oh no, error.")
	}

	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalln(err)
	}
	responseString := string(b)

	println(response)
	return responseString
}
