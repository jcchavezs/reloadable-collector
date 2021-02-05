package aggregatedprocessor

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.uber.org/zap"
)

var _ component.Processor = (*processor)(nil)

type processor struct {
	logger            *zap.Logger
	host              component.Host
	subprocessors     []component.TracesProcessor
	firstSubprocessor consumer.TracesConsumer
	mx                *sync.Mutex
}

// GetCapabilities must return the capabilities of the processor.
func (p *processor) GetCapabilities() component.ProcessorCapabilities {
	for _, sp := range p.subprocessors {
		if sp.GetCapabilities().MutatesConsumedData {
			return component.ProcessorCapabilities{MutatesConsumedData: true}
		}
	}

	return component.ProcessorCapabilities{}
}

// Start tells the component to start.
func (p *processor) Start(ctx context.Context, host component.Host) error {
	p.host = host
	return p.start(ctx, host)
}

func (p *processor) start(ctx context.Context, host component.Host) error {
	for i := len(p.subprocessors) - 1; i >= 0; i-- {
		if err := p.subprocessors[i].Start(ctx, host); err != nil {
			return err
		}
	}
	return nil
}

// Shutdown is invoked during service shutdown.
func (p *processor) Shutdown(ctx context.Context) error {
	for _, subprocessor := range p.subprocessors {
		if err := subprocessor.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ConsumeTraces receives pdata.Traces for processing.
func (p *processor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	return p.firstSubprocessor.ConsumeTraces(ctx, td)
}

func (p *processor) replaceSubprocessors(ctx context.Context, subprocessors []component.TracesProcessor) error {
	p.logger.Debug("Replacing subprocessors")
	oldSubprocessors := p.subprocessors

	p.mx.Lock()
	defer p.mx.Unlock()
	p.subprocessors = subprocessors
	p.firstSubprocessor = subprocessors[0]

	err := p.start(ctx, p.host)
	if err != nil {
		return err
	}

	go func(subprocessors []component.TracesProcessor, logger *zap.Logger) {
		logger.Debug("Shutting down old subprocessors")
		for _, p := range subprocessors {
			if err := p.Shutdown(ctx); err != nil {
				logger.Sugar().Errorf("failed to shutdown old subprocessor: %v", err)
			}
		}
	}(oldSubprocessors, p.logger)

	return nil
}
