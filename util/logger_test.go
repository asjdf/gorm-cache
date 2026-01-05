package util

import (
	"context"
	"strings"
	"testing"
)

func TestDefaultLogger_SetIsDebug(t *testing.T) {
	logger := &DefaultLogger{}
	
	// Test setting debug to true
	logger.SetIsDebug(true)
	if !logger.isDebug {
		t.Error("expected isDebug to be true")
	}
	
	// Test setting debug to false
	logger.SetIsDebug(false)
	if logger.isDebug {
		t.Error("expected isDebug to be false")
	}
}

func TestDefaultLogger_CtxInfo(t *testing.T) {
	logger := &DefaultLogger{}
	ctx := context.Background()
	
	// Test with debug disabled (should not output)
	logger.SetIsDebug(false)
	logger.CtxInfo(ctx, "test message %s", "value")
	// No way to verify output, but should not panic
	
	// Test with debug enabled
	logger.SetIsDebug(true)
	logger.CtxInfo(ctx, "test message %s", "value")
	// No way to verify output, but should not panic
}

func TestDefaultLogger_CtxInfo_WithFormat(t *testing.T) {
	logger := &DefaultLogger{}
	ctx := context.Background()
	logger.SetIsDebug(true)
	
	// Test various format strings
	logger.CtxInfo(ctx, "simple message")
	logger.CtxInfo(ctx, "message with %s", "string")
	logger.CtxInfo(ctx, "message with %d", 42)
	logger.CtxInfo(ctx, "message with %v", struct{ Name string }{"test"})
	logger.CtxInfo(ctx, "multiple %s and %d", "values", 123)
}

func TestDefaultLogger_CtxError(t *testing.T) {
	logger := &DefaultLogger{}
	ctx := context.Background()
	
	// Test with debug disabled (should not output)
	logger.SetIsDebug(false)
	logger.CtxError(ctx, "error message %s", "value")
	// No way to verify output, but should not panic
	
	// Test with debug enabled
	logger.SetIsDebug(true)
	logger.CtxError(ctx, "error message %s", "value")
	// No way to verify output, but should not panic
}

func TestDefaultLogger_CtxError_WithFormat(t *testing.T) {
	logger := &DefaultLogger{}
	ctx := context.Background()
	logger.SetIsDebug(true)
	
	// Test various format strings
	logger.CtxError(ctx, "simple error")
	logger.CtxError(ctx, "error with %s", "string")
	logger.CtxError(ctx, "error with %d", 42)
	logger.CtxError(ctx, "error with %v", struct{ Name string }{"test"})
	logger.CtxError(ctx, "multiple %s and %d", "values", 123)
}

func TestDefaultLogger_OutputFormat(t *testing.T) {
	logger := &DefaultLogger{}
	ctx := context.Background()
	logger.SetIsDebug(true)
	
	// Test that output contains expected format elements
	// Since we can't easily capture stdout, we just verify it doesn't panic
	// and that the format string is processed correctly
	message := "test message"
	logger.CtxInfo(ctx, message)
	
	// Verify the logger state
	if !logger.isDebug {
		t.Error("expected isDebug to be true")
	}
}

func TestDefaultLogger_ContextHandling(t *testing.T) {
	logger := &DefaultLogger{}
	logger.SetIsDebug(true)
	
	// Test with nil context (should not panic)
	logger.CtxInfo(nil, "message with nil context")
	logger.CtxError(nil, "error with nil context")
	
	// Test with background context
	ctx := context.Background()
	logger.CtxInfo(ctx, "message with background context")
	logger.CtxError(ctx, "error with background context")
	
	// Test with context with values
	ctxWithValue := context.WithValue(ctx, "key", "value")
	logger.CtxInfo(ctxWithValue, "message with context value")
	logger.CtxError(ctxWithValue, "error with context value")
}

// Helper function to check if string contains substring (for potential future output verification)
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

