package processor2

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

var _ component.TracesProcessor = (*processor)(nil)

type processor struct {
	next consumer.TracesConsumer
}

func newProcessor2(next consumer.TracesConsumer) *processor {
	return &processor{next: next}
}

func (s *processor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	fmt.Println("Call to P2")
	return s.next.ConsumeTraces(ctx, td)
}

func (s *processor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

func (s *processor) Start(_ context.Context, host component.Host) error {
	return nil
}

func (s *processor) Shutdown(context.Context) error {
	fmt.Println("calling shutdown on processor 2 :(")
	return nil
}
