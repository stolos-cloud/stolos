package marshal

import (
	"os"

	"github.com/goccy/go-json"
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

func SaveStateToJSON(saveState state.SaveState) error {
	jsonData, err := json.Marshal(saveState)
	if err != nil {
		return err
	}
	err = os.WriteFile("bootstrap-state.json", jsonData, 0644)
	if err != nil {
		return err
	}
	err = talos.SaveSplitConfigBundleFiles(*state.ConfigBundle)
	if err != nil {
		return err
	}
	return nil
}

func ReadStateFromJSON() state.SaveState {
	var saveState state.SaveState
	stateFile, err := os.ReadFile("bootstrap-state.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(stateFile, &saveState)
	if err != nil {
		panic(err)
	}
	state.ConfigBundle, err = talos.ReadSplitConfigBundleFiles()
	return saveState
}
