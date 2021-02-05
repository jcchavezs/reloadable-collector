package processor1

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/pdata"
)

var _ component.TracesProcessor = (*processor)(nil)

type processor struct {
	phrase string
	next   consumer.TracesConsumer
}

func newProcessor1(next consumer.TracesConsumer, phrase string) *processor {
	return &processor{next: next, phrase: phrase}
}

func (s *processor) ConsumeTraces(ctx context.Context, td pdata.Traces) error {
	fmt.Printf("Call to P1: Phrase: %s\n", s.phrase)
	return s.next.ConsumeTraces(ctx, td)
}

func (s *processor) GetCapabilities() component.ProcessorCapabilities {
	return component.ProcessorCapabilities{MutatesConsumedData: true}
}

func (s *processor) Start(_ context.Context, host component.Host) error {
	return nil
}

func (s *processor) Shutdown(context.Context) error {
	fmt.Println("calling shutdown on processor 1 :(")
	return nil
}
