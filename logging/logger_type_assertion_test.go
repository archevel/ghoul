package logging

import (
	"log/slog"
	"testing"
)

func TestLevelReplacerTypeAssertionSafety(t *testing.T) {
	// Test that levelReplacer doesn't panic with unexpected attribute values
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("levelReplacer should not panic, got: %v", r)
		}
	}()

	// Test with valid slog.Level
	validAttr := slog.Attr{
		Key:   slog.LevelKey,
		Value: slog.AnyValue(LevelTrace),
	}
	result := levelReplacer(nil, validAttr)
	if result.Value.String() != "TRACE" {
		t.Errorf("Expected TRACE, got %s", result.Value.String())
	}

	// Test with invalid type (this could potentially cause panic)
	invalidAttr := slog.Attr{
		Key:   slog.LevelKey,
		Value: slog.StringValue("not-a-level"), // Wrong type
	}

	// This should not panic
	result = levelReplacer(nil, invalidAttr)
	// Should return the original attribute unchanged if not a valid level
}

func TestLevelReplacerWithNonLevelAttribute(t *testing.T) {
	// Test that non-level attributes are passed through unchanged
	attr := slog.Attr{
		Key:   "other-key",
		Value: slog.StringValue("some-value"),
	}

	result := levelReplacer(nil, attr)
	if result.Key != attr.Key || result.Value.String() != attr.Value.String() {
		t.Error("Non-level attributes should be passed through unchanged")
	}
}