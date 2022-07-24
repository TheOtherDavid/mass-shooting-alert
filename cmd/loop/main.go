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
		interval, err := strconv.Atoi(os.Getenv("INTERVAL_SECONDS"))
		if err != nil {
			fmt.Printf("INTERVAL_SECONDS environment variable must be an integer.\n")
			return
		}
		fmt.Printf("Executing Gun Violence Alert.\n")
		alert.GunViolenceAlert()
		fmt.Printf("Gun Violence Alert complete. Sleeping %d seconds.\n", interval)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
