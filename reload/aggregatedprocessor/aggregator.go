package aggregatedprocessor

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/zap"
)

type processorSettings struct {
	*processor
	nextConsumer consumer.TracesConsumer
	params       component.ProcessorCreateParams
	pipeline     []string
	factories    map[configmodels.Type]component.ProcessorFactory
	config       *config
}

// Aggregator is in charge of the lifecycle of the aggregated processors
type Aggregator struct {
	logger               *zap.Logger
	aggregatedprocessors []processorSettings
	processorFactories   map[configmodels.Type]component.ProcessorFactory
	configUpdate         chan configmodels.Processors
}

// New creates a new aggregator
func New(fs map[configmodels.Type]component.ProcessorFactory) *Aggregator {
	a := &Aggregator{
		processorFactories: fs,
		configUpdate:       make(chan configmodels.Processors),
	}
	go a.listenToConfigUpdate()
	return a
}

func (p *Aggregator) listenToConfigUpdate() {
	for {
		c := <-p.configUpdate
		if len(c) == 0 {
			// channel was closed.
			return
		}

		p.logger.Debug("Received a configuration update")
		err := p.ReloadProcessors(context.Background(), c)
		if err != nil {
			p.logger.Sugar().Errorf("Failed to update configuration for processors: %v", err)
		}
	}
}

// ConfigUpdate returns the channel to send the config updates
func (p *Aggregator) ConfigUpdate() chan configmodels.Processors {
	return p.configUpdate
}

// CreateTraceProcessor creates a new aggregated processor as part of the factory
func (p *Aggregator) CreateTraceProcessor(
	ctx context.Context,
	params component.ProcessorCreateParams,
	cfg configmodels.Processor,
	nextConsumer consumer.TracesConsumer,
) (component.TracesProcessor, error) {
	aggregatedCfg := cfg.(*config)

	subprocessors, err := p.buildSubprocessors(ctx, params, aggregatedCfg.subprocessorsConfigs, nextConsumer)
	if err != nil {
		return nil, err
	}

	ap := &processor{
		subprocessors:     subprocessors,
		firstSubprocessor: subprocessors[0],
		logger:            params.Logger,
		mx:                &sync.Mutex{},
	}

	if p.logger == nil {
		p.logger = params.Logger
	}

	p.aggregatedprocessors = append(p.aggregatedprocessors, processorSettings{
		processor:    ap,
		nextConsumer: nextConsumer,
		params:       params,
		pipeline:     aggregatedCfg.pipeline,
		config:       aggregatedCfg,
	})

	return ap, nil
}

func (p *Aggregator) buildSubprocessors(
	ctx context.Context,
	params component.ProcessorCreateParams,
	cfgs []subprocessorConfig,
	nextConsumer consumer.TracesConsumer,
) ([]component.TracesProcessor, error) {
	var (
		nc  = nextConsumer
		err error
	)

	subprocessors := make([]component.TracesProcessor, len(cfgs))
	for i := len(cfgs) - 1; i >= 0; i-- {
		subCfg := cfgs[i]
		factory := p.processorFactories[configmodels.Type(subCfg.factoryName)]
		nc, err = factory.CreateTracesProcessor(ctx, params, subCfg.config, nc)
		if err != nil {
			return nil, fmt.Errorf("failed to create subprocessor: %v", err)
		}
		subprocessors[i] = nc.(component.TracesProcessor)
	}
	return subprocessors, nil
}

// Factory creates a processor factory for an aggregated processor
func (p *Aggregator) Factory() component.ProcessorFactory {
	return processorhelper.NewFactory(
		Type,
		CreateDefaultConfig,
		processorhelper.WithTraces(p.CreateTraceProcessor),
	)
}

// Type returns the for an aggregated processor
func (p *Aggregator) Type() configmodels.Type {
	return Type
}

// AggregateConfig aggregates the config for all the processor in a pipeline to be
// passed to the aggregated processor factory
func (p *Aggregator) AggregateConfig(
	pipelineProcessors []string,
	subprocessorsConfig configmodels.Processors,
) configmodels.Processor {
	cfg := &config{
		ProcessorSettings: configmodels.ProcessorSettings{
			TypeVal: Type,
			NameVal: Type,
		},
		pipeline:             pipelineProcessors,
		subprocessorsConfigs: make([]subprocessorConfig, len(pipelineProcessors)),
	}

	for i, p := range pipelineProcessors {
		cfg.subprocessorsConfigs[i] = subprocessorConfig{
			name:        p,
			factoryName: strings.SplitN(p, "/", 2)[0],
			config:      subprocessorsConfig[p],
		}
	}

	return cfg
}

// ReloadProcessors reloads the aggregated processors and
func (p *Aggregator) ReloadProcessors(ctx context.Context, newCfgs configmodels.Processors) error {
	for _, aps := range p.aggregatedprocessors {
		for name, newCfg := range newCfgs {
			for i := 0; i < len(aps.config.subprocessorsConfigs); i++ {
				if aps.config.subprocessorsConfigs[i].name == name {
					aps.config.subprocessorsConfigs[i].config = newCfg
				}
			}
		}

		subprocessors, err := p.buildSubprocessors(ctx, aps.params, aps.config.subprocessorsConfigs, aps.nextConsumer)
		if err != nil {
			return err
		}

		err = aps.replaceSubprocessors(ctx, subprocessors)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetProcessorName returns the aggregated processor name based on the pipeline name
func GetProcessorName(pipelineName string) string {
	return Type + "_" + pipelineName
}
