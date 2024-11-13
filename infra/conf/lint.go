package conf

import "errors"

type ConfigureFilePostProcessingStage interface {
	Process(conf *Config) error
}

var configureFilePostProcessingStages map[string]ConfigureFilePostProcessingStage

func RegisterConfigureFilePostProcessingStage(name string, stage ConfigureFilePostProcessingStage) {
	if configureFilePostProcessingStages == nil {
		configureFilePostProcessingStages = make(map[string]ConfigureFilePostProcessingStage)
	}
	configureFilePostProcessingStages[name] = stage
}

func PostProcessConfigureFile(conf *Config) error {
	for _, v := range configureFilePostProcessingStages {
		// 检查 sniffing 及 dns 设置
		if err := v.Process(conf); err != nil {
			return errors.New("fejected by Postprocessing Stage")
		}
	}
	return nil
}
