package marshal

import (
	"os"

	"github.com/goccy/go-json"
	"github.com/stolos-cloud/stolos-bootstrap/internal/tui"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/state"
	"github.com/stolos-cloud/stolos-bootstrap/pkg/talos"
)

func ReadBootstrapInfos(filename string, bootstrapInfos *state.BootstrapInfo) {
	configFile, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(configFile, &bootstrapInfos)
	if err != nil {
		panic(err)
	}
}

func SaveStateToJSON(logger *tui.UILogger, saveState state.SaveState) {
	jsonData, err := json.Marshal(saveState)
	if err != nil {
		logger.Errorf("Error saving state to JSON: %v\n", err)
		return
	}
	err = os.WriteFile("bootstrap-state.json", jsonData, 0644)
	if err != nil {
		logger.Errorf("Error saving state to JSON: %v\n", err)
		return
	}
	err = talos.SaveSplitConfigBundleFiles(*state.ConfigBundle)
	if err != nil {
		logger.Errorf("Error saving state to JSON: %v\n", err)
		return
	}
}

func ReadStateFromJSON(saveState state.SaveState, bootstrapInfos *state.BootstrapInfo) {
	stateFile, err := os.ReadFile("bootstrap-state.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(stateFile, &saveState)
	if err != nil {
		panic(err)
	}
	bootstrapInfos = &saveState.BootstrapInfo
	state.ConfigBundle, err = talos.ReadSplitConfigBundleFiles()
	if err != nil {
		panic(err)
	}
}
