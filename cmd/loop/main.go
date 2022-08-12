package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/TheOtherDavid/mass-shooting-alert"
)

func main() {
	for {
		interval, err := strconv.Atoi(os.Getenv("LOOP_INTERVAL_SECONDS"))
		if err != nil {
			fmt.Printf("LOOP_INTERVAL_SECONDS environment variable must be an integer.\n")
			return
		}
		fmt.Printf("Executing Mass Shooting Alert at %s\n", time.Now())
		alert.MassShootingAlert()
		fmt.Printf("Mass Shooting Alert complete. Sleeping %d seconds.\n\n", interval)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
