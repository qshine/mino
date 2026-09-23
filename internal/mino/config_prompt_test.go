package mino

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestConfigInputPreservesBufferedChat(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("fake-key\n你好\n"))
	value, err := readConfigLine(context.Background(), reader)
	if err != nil || value != "fake-key\n" {
		t.Fatalf("value = %q, error = %v", value, err)
	}
	remaining, err := io.ReadAll(reader)
	if err != nil || string(remaining) != "你好\n" {
		t.Fatal("configuration consumed the first chat question")
	}
}

func TestConfigInputCancellation(t *testing.T) {
	input, writer := io.Pipe()
	defer input.Close()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := readConfigLine(ctx, bufio.NewReader(input)); done <- err }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("configuration did not stop on cancellation")
	}
}

func TestConfigInputEOF(t *testing.T) {
	if _, err := readConfigLine(context.Background(), bufio.NewReader(strings.NewReader(""))); !errors.Is(err, io.EOF) {
		t.Fatalf("EOF error = %v", err)
	}
}
