package alert

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/ini.v1"
)

func MassShootingAlert() {
	//Find last triggered date.
	//Access local data file?

	lastShootingCity, lastShootingDate, lastTriggeredDate, err := getLastTriggeredData()
	if err != nil {
		fmt.Printf("Error retrieving data from file.\n")
		return
	}
	//Hit the MST bucket and get the last updated date
	lastUpdatedDate, err := queryS3Bucket()
	if err != nil {
		fmt.Printf("Error retrieving metadata from S3 bucket: %s\n", err)
		return
	}

	var incidents []Incident

	//If the last updated date is NEWER than the last triggered date, download the file
	//TODO: Pull this out into its own function.
	if lastTriggeredDate.Before(lastUpdatedDate) {
		fmt.Printf("Last alert triggered date %s is before last file update date %s. Downloading incidents.\n", lastTriggeredDate.String(), lastUpdatedDate.String())

		incidents, err = getIncidents()
		if err != nil {
			fmt.Printf("Error retrieving incidents from S3 bucket: %s\n", err)
			return
		}
		fmt.Printf("Incidents downloaded.\n")
	} else {
		fmt.Printf("Last alert triggered date %s is after last file update date %s. Not downloading incidents.\n", lastTriggeredDate.String(), lastUpdatedDate.String())
		fmt.Printf("No shootings this time!\n")
		return
	}

	newShooting := isNewShootingToday(incidents, lastShootingCity, lastShootingDate)

	//If result is true, call WLED
	if newShooting {
		fmt.Printf("Oh no, there's a new shooting!\n")
		//Calculations
		//Maybe find the total number of dead/wounded, and compare it to a high mark like 50 to make the pulse different speeds?
		dead, wounded := extractDailyDeadAndWoundedCount(incidents)
		victims := dead + wounded

		victimThreshold, _ := strconv.Atoi(os.Getenv("VICTIM_THRESHOLD"))

		//Set a threshold, to only activate for a minimum number of victims
		if victims > victimThreshold {
			fmt.Printf(strconv.Itoa(victims) + "victims!\n")
			//Do some other stuff
			//Should we make this a goroutine? That way we don't have to wait on it to update the file
			err = SendWLEDPulse()
			if err != nil {
				fmt.Printf("Error sending WLED pulse: %s\n", err)
			}
		} else {
			fmt.Printf("Not enough victims to trigger alert.\n")
		}
		//Update the data file to have the latest data
		lastTriggeredIncident := incidents[0]
		lastShootingCity = lastTriggeredIncident.City
		lastShootingDate = lastTriggeredIncident.Date

		lastTriggeredDate = time.Now().UTC()
		SetLastTriggeredData(lastShootingCity, lastShootingDate, lastTriggeredDate)
	} else {
		zeroTime := time.Time{}
		now := time.Now().UTC()
		fmt.Printf("Updating last triggered data with date %s\n", now)
		SetLastTriggeredData("", zeroTime, now)
		fmt.Printf("No shootings this time!\n")
	}

}

func getLastTriggeredData() (lastShootingCity string, lastShootingDate time.Time, lastTriggeredDate time.Time, err error) {
	appRoot := os.Getenv("APP_ROOT")
	cfg, err := ini.Load(appRoot + "/config/data.ini")
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

	return lastShootingCity, lastShootingDate, lastTriggeredDate, nil
}

func SetLastTriggeredData(lastShootingCity string, lastShootingDate time.Time, lastTriggeredDate time.Time) {
	appRoot := os.Getenv("APP_ROOT")
	filename := appRoot + "/config/data.ini"

	cfg, err := ini.Load(filename)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	layout := "2006-01-02T15:04:05.000Z"
	zeroTime := time.Time{}

	if lastShootingCity != "" {
		cfg.Section("").Key("last_shooting_city").SetValue(lastShootingCity)
	}
	if lastShootingDate != zeroTime {
		lastShootingDateString := lastShootingDate.Format(layout)
		cfg.Section("").Key("last_shooting_date").SetValue(lastShootingDateString)
	}
	if lastTriggeredDate != zeroTime {
		lastTriggeredDateString := lastTriggeredDate.Format(layout)
		cfg.Section("").Key("last_triggered_date").SetValue(lastTriggeredDateString)
	}

	cfg.SaveTo(filename)
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
	previousIncident := incidents[0]
	dead, _ := strconv.Atoi(previousIncident.Killed)
	wounded, _ := strconv.Atoi(previousIncident.Wounded)

	return dead, wounded
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
	fmt.Printf("Analysing incidents.\n")
	//Determine whether there has been a shooting that meets the criteria
	//Date/City is close enough, since we don't have a real timestamp. Unlikely for multiple shootings in the same city on the same day.
	if len(incidents) == 0 {
		return false
	}
	//Since the most recent shooting is always first, we only need to look at the first shooting.
	incident := incidents[0]
	if incident.City != lastShootingCity || incident.Date != lastShootingDate {
		return true
	}
	return false
}
