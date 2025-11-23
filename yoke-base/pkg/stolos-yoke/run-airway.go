//go:build airway

package stolos_yoke

import (
	"flag"
	"os"
)

func Run[flightResource any](inputs AirwayInputs, renderTemplateFunc func() ([]byte, error)) {

	flightUrl := flag.String("flight-url", "", "flight url")
	flag.Parse()

	if *flightUrl == "" {
		panic("flight url is required")
	}

	airway, err := BuildAirwayFor[flightResource](inputs, *flightUrl)

	if err != nil {
		panic(err)
	}
	if _, err := os.Stdout.Write(airway); err != nil {
		panic(err)
	}
}
