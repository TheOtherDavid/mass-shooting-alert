package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/TheOtherDavid/gun-violence-alert"
)

func main() {
	for {
		interval, err := strconv.Atoi(os.Getenv("LOOP_INTERVAL_SECONDS"))
		if err != nil {
			fmt.Printf("LOOP_INTERVAL_SECONDS environment variable must be an integer.\n")
			return
		}
		fmt.Printf("Executing Gun Violence Alert at %s\n", time.Now())
		alert.GunViolenceAlert()
		fmt.Printf("Gun Violence Alert complete. Sleeping %d seconds.\n\n", interval)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
