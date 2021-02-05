package reload

import (
	"github.com/jcchavezs/reload-collector/reload/aggregatedprocessor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/service"
)

func prepareConfigFactory(delegate service.ConfigFactory, aggregator *aggregatedprocessor.Aggregator) service.ConfigFactory {
	return func(v *viper.Viper, cmd *cobra.Command, factories component.Factories) (*configmodels.Config, error) {
		cfg, err := delegate(v, cmd, factories)
		if err != nil {
			return nil, err
		}

		for key, pipeline := range cfg.Pipelines {
			// return the aggregated processor name
			pipelineProcessorName := aggregatedprocessor.GetProcessorName(key)
			newPipeline := configmodels.Pipeline{
				Name:      pipeline.Name,
				InputType: pipeline.InputType,
				Receivers: pipeline.Receivers,
				// the aggregated processor takes over the pipeline
				Processors: []string{pipelineProcessorName},
				Exporters:  pipeline.Exporters,
			}
			cfg.Pipelines[key] = &newPipeline
			// add the aggregated procesor to the list of processors
			cfg.Processors[pipelineProcessorName] = aggregator.AggregateConfig(pipeline.Processors, cfg.Processors)
		}

		return cfg, err
	}
}

// PrepareServiceParameters does the needed changes for the aggregated processor to take
// over the pipelines and returns a channel where the config updates can be sent to.
func PrepareServiceParameters(p service.Parameters) (service.Parameters, chan configmodels.Processors) {
	configFactory := p.ConfigFactory
	if configFactory == nil {
		// use default factory that loads the configuration file
		configFactory = service.FileLoaderConfigFactory
	}

	aggregator := aggregatedprocessor.New(p.Factories.Processors)

	params := service.Parameters{
		ApplicationStartInfo: p.ApplicationStartInfo,
		Factories:            p.Factories,
		ConfigFactory:        prepareConfigFactory(configFactory, aggregator),
		LoggingOptions:       p.LoggingOptions,
	}

	// adds the aggregated processor type to the list of factories
	params.Factories.Processors[aggregatedprocessor.Type] = aggregator.Factory()

	return params, aggregator.ConfigUpdate()
}
