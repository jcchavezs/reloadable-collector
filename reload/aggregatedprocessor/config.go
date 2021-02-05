package aggregatedprocessor

import (
	"go.opentelemetry.io/collector/config/configmodels"
)

// Type is the name of the aggregated processor
const Type = "aggregatedprocessor"

type subprocessorConfig struct {
	name        string
	factoryName string
	config      configmodels.Processor
}

type config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`
	pipeline                       []string
	subprocessorsConfigs           []subprocessorConfig
}

// CreateDefaultConfig creates the default config for aggregated processor
func CreateDefaultConfig() configmodels.Processor {
	return &config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: Type,
			NameVal: Type,
		},
	}
}
