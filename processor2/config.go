package processor2

import "go.opentelemetry.io/collector/config/configmodels"

type Config struct {
	configmodels.ProcessorSettings `mapstructure:",squash"`
}
