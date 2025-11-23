//go:build !airway

package stolos_yoke

import (
	"os"
)

func Run[flightResource any](inputs AirwayInputs, renderTemplateFunc func() ([]byte, error)) {

	rendered, err := renderTemplateFunc()
	if err != nil {
		panic(err)
	}
	if _, err := os.Stdout.Write(rendered); err != nil {
		panic(err)
	}
}
