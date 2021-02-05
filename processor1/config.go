package processor1

import "go.opentelemetry.io/collector/config/configmodels"

type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`
	Phrase                         string `mapstructure:"phrase"`
}
