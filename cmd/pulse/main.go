package main

import (
	"fmt"
	"time"

	"github.com/TheOtherDavid/mass-shooting-alert"
)

func main() {
	for {
		fmt.Printf("Executing WLED Pulse at %s\n", time.Now())
		alert.SendWLEDPulse()
	}
}
