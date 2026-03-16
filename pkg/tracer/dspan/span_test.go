package dspan

import (
	"context"
	"errors"
	"testing"

	"github.com/dehwyy/tracerfx/pkg/tracer/log"
)

func TestSpanStartAndEnd(t *testing.T) {
	ctx := context.Background()
	logger := log.NewZerologLogger()
	ctx = log.ContextWithLogger(ctx, logger)

	ctxTrace, span := Start(ctx, "test_span", Attr("test_key", "test_value"))
	if ctxTrace == nil {
		t.Fatal("context is nil")
	}
	if span == nil {
		t.Fatal("span is nil")
	}

	span.WithAttribute("custom_key", "custom_value")

	type TestStruct struct {
		Name string
		Age  int
		priv string
	}

	span.WithAttribute("my_struct", TestStruct{
		Name: "Alice",
		Age:  30,
		priv: "secret",
	})

	span.End()
}

func TestSpanErr(t *testing.T) {
	ctx := context.Background()
	_, span := Start(ctx, "test_span", Attr("error_context", "test_stage"))

	err := errors.New("test error")
	if span.Err(err) != err {
		t.Fatal("expected error to be returned")
	}
}

func TestExtractFields(t *testing.T) {
	type Nested struct {
		Value int
	}
	type MyStruct struct {
		PublicField  string
		NestedStruct Nested
		unexported   bool
	}

	data := MyStruct{
		PublicField: "hello",
		NestedStruct: Nested{
			Value: 42,
		},
		unexported: true,
	}

	extracted := extractFields("data", data)

	if extracted["data.PublicField"] != "hello" {
		t.Errorf("expected data.PublicField = 'hello', got %v", extracted["data.PublicField"])
	}

	if extracted["data.NestedStruct.Value"] != 42 {
		t.Errorf("expected data.NestedStruct.Value = 42, got %v", extracted["data.NestedStruct.Value"])
	}

	if _, ok := extracted["data.unexported"]; ok {
		t.Errorf("did not expect unexported field to be extracted")
	}
}

func TestAddAttributeToSpanConcurrency(t *testing.T) {
	ctx := context.Background()
	_, span := Start(ctx, "test_span", Attr("test_key", "test_value"))

	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(idx int) {
			span.WithAttribute("key", idx)
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	span.End()
}
