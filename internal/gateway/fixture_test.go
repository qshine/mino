package gateway

import (
	"context"
	"io"

	"github.com/qshine/mino/internal/agent"
)

type fakeHandler func(context.Context, string, func(string) error) error

func (f fakeHandler) Start(context.Context, agent.Interaction) error { return nil }
func (f fakeHandler) Handle(ctx context.Context, message string, i agent.Interaction) error {
	if err := i.Emit(agent.Event{Kind: "response_started"}); err != nil {
		return err
	}
	return f(ctx, message, func(text string) error { return i.Emit(agent.Event{Kind: "text", Text: text}) })
}
func runTerminal(ctx context.Context, input io.Reader, output, errorOutput io.Writer, respond func(context.Context, string, func(string) error) error) error {
	return NewCLI(input, output, errorOutput).Run(ctx, fakeHandler(respond))
}
func runTerminalLines(ctx context.Context, lines <-chan inputLine, output, errorOutput io.Writer, respond func(context.Context, string, func(string) error) error) error {
	// Keep one shared input stream for chat and confirmation in these focused tests.
	c := NewCLI(nil, output, errorOutput)
	c.lines = lines
	return c.runLines(ctx, fakeHandler(respond))
}
