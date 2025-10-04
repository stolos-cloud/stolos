package marshal

import (
	"os"

	"github.com/goccy/go-json"
	"github.com/siderolabs/talos/pkg/machinery/config/bundle"
)

func UnmarshalFromFile[T interface{}](filename string) (T, error) {
	object := new(T)
	configFile, err := os.ReadFile(filename)
	if err != nil {
		return *object, err
	}
	err = json.Unmarshal(configFile, object)
	return *object, nil
}

func MarshalToFile(filename string, object interface{}) error {
	jsonData, err := json.Marshal(object)
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, jsonData, 0644)
	return err
}

// SaveSplitConfigBundleFiles take a config bundle and saves each composite part to individual files for later loading
func SaveSplitConfigBundleFiles(configBundle *bundle.Bundle) error {
	initBytes, err := configBundle.InitCfg.Bytes()
	err = os.WriteFile("init.yaml", initBytes, 0644)
	workerBytes, err := configBundle.WorkerCfg.Bytes()
	err = os.WriteFile("worker.yaml", workerBytes, 0644)
	controlPlaneBytes, err := configBundle.ControlPlaneCfg.Bytes()
	err = os.WriteFile("controlplane.yaml", controlPlaneBytes, 0644)
	talosBytes, err := configBundle.TalosCfg.Bytes()
	err = os.WriteFile("talosconfig", talosBytes, 0644)
	return err
}

// ReadSplitConfigBundleFiles reconstructs multiple yaml configs into a ConfigBundle
func ReadSplitConfigBundleFiles() (*bundle.Bundle, error) {
	configBundleOpts := []bundle.Option{
		bundle.WithExistingConfigs("./"),
	}

	return bundle.NewBundle(configBundleOpts...)
}

//func SaveStateToJSON(saveState state.SaveState) error {
//	jsonData, err := json.Marshal(saveState)
//	if err != nil {
//		return err
//	}
//	err = os.WriteFile("bootstrap-state.json", jsonData, 0644)
//	if err != nil {
//		return err
//	}
//	if state.ConfigBundle != nil {
//		err = SaveSplitConfigBundleFiles(state.ConfigBundle)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

//func ReadStateFromJSON() state.SaveState {
//	var saveState state.SaveState
//	stateFile, err := os.ReadFile("bootstrap-state.json")
//	if err != nil {
//		panic(err)
//	}
//	err = json.Unmarshal(stateFile, &saveState)
//	if err != nil {
//		panic(err)
//	}
//	state.ConfigBundle, err = ReadSplitConfigBundleFiles()
//	return saveState
//}
