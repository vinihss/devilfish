package valueobject_test

import (
	"testing"

	"devilfish/internal/domain/valueobject"
)

func TestNewMessageContent_WithValidText_ReturnsContent(t *testing.T) {
	content, err := valueobject.NewMessageContent("Hello, World!")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content == nil {
		t.Fatal("expected content, got nil")
	}
	if content.Text != "Hello, World!" {
		t.Errorf("expected Hello, World!, got %s", content.Text)
	}
	if content.HTML != "" {
		t.Errorf("expected empty HTML, got %s", content.HTML)
	}
	if content.Markdown != "" {
		t.Errorf("expected empty Markdown, got %s", content.Markdown)
	}
}

func TestNewMessageContent_WithEmptyText_ReturnsError(t *testing.T) {
	content, err := valueobject.NewMessageContent("")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if content != nil {
		t.Errorf("expected nil content, got %v", content)
	}
	if err != valueobject.ErrEmptyContent {
		t.Errorf("expected ErrEmptyContent, got %v", err)
	}
}

func TestNewMessageContentWithFormats_WithValidInput_ReturnsContent(t *testing.T) {
	content, err := valueobject.NewMessageContentWithFormats(
		"Hello, World!",
		"<p>Hello, World!</p>",
		"**Hello, World!**",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content == nil {
		t.Fatal("expected content, got nil")
	}
	if content.Text != "Hello, World!" {
		t.Errorf("expected Hello, World!, got %s", content.Text)
	}
	if content.HTML != "<p>Hello, World!</p>" {
		t.Errorf("expected HTML, got %s", content.HTML)
	}
	if content.Markdown != "**Hello, World!**" {
		t.Errorf("expected Markdown, got %s", content.Markdown)
	}
}

func TestNewMessageContentWithFormats_WithEmptyText_ReturnsError(t *testing.T) {
	content, err := valueobject.NewMessageContentWithFormats("", "<p>Hello</p>", "**Hello**")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if content != nil {
		t.Errorf("expected nil content, got %v", content)
	}
	if err != valueobject.ErrEmptyContent {
		t.Errorf("expected ErrEmptyContent, got %v", err)
	}
}

func TestMessageContent_Equals_WithEqualContent_ReturnsTrue(t *testing.T) {
	mc1, _ := valueobject.NewMessageContentWithFormats("Hello", "<p>Hello</p>", "**Hello**")
	mc2, _ := valueobject.NewMessageContentWithFormats("Hello", "<p>Hello</p>", "**Hello**")

	if !mc1.Equals(mc2) {
		t.Error("expected content to be equal")
	}
}

func TestMessageContent_Equals_WithDifferentText_ReturnsFalse(t *testing.T) {
	mc1, _ := valueobject.NewMessageContent("Hello")
	mc2, _ := valueobject.NewMessageContent("World")

	if mc1.Equals(mc2) {
		t.Error("expected content to not be equal")
	}
}

func TestMessageContent_Equals_WithDifferentFormats_ReturnsFalse(t *testing.T) {
	mc1, _ := valueobject.NewMessageContentWithFormats("Hello", "<p>Hello</p>", "")
	mc2, _ := valueobject.NewMessageContentWithFormats("Hello", "", "**Hello**")

	if mc1.Equals(mc2) {
		t.Error("expected content to not be equal")
	}
}

func TestMessageContent_Equals_WithNilOther_ReturnsFalse(t *testing.T) {
	mc, _ := valueobject.NewMessageContent("Hello")

	if mc.Equals(nil) {
		t.Error("expected comparison to return false")
	}
}

func TestMessageContent_Equals_WhenNilCompared_ReturnsTrue(t *testing.T) {
	var mc *valueobject.MessageContent

	if !mc.Equals(nil) {
		t.Error("expected comparison to return true")
	}
}

func TestMessageContent_Equals_BothNil_ReturnsTrue(t *testing.T) {
	var mc1, mc2 *valueobject.MessageContent

	if !mc1.Equals(mc2) {
		t.Error("expected comparison to return true")
	}
}

func TestMessageContent_HasHTML_WithHTML_ReturnsTrue(t *testing.T) {
	mc, _ := valueobject.NewMessageContentWithFormats("Hello", "<p>Hello</p>", "")

	if !mc.HasHTML() {
		t.Error("expected HasHTML to return true")
	}
}

func TestMessageContent_HasHTML_WithoutHTML_ReturnsFalse(t *testing.T) {
	mc, _ := valueobject.NewMessageContent("Hello")

	if mc.HasHTML() {
		t.Error("expected HasHTML to return false")
	}
}

func TestMessageContent_HasMarkdown_WithMarkdown_ReturnsTrue(t *testing.T) {
	mc, _ := valueobject.NewMessageContentWithFormats("Hello", "", "**Hello**")

	if !mc.HasMarkdown() {
		t.Error("expected HasMarkdown to return true")
	}
}

func TestMessageContent_HasMarkdown_WithoutMarkdown_ReturnsFalse(t *testing.T) {
	mc, _ := valueobject.NewMessageContent("Hello")

	if mc.HasMarkdown() {
		t.Error("expected HasMarkdown to return false")
	}
}

func TestMessageContent_Length_ReturnsTextLength(t *testing.T) {
	mc, _ := valueobject.NewMessageContent("Hello, World!")

	if mc.Length() != 13 {
		t.Errorf("expected 13, got %d", mc.Length())
	}
}

func TestMessageContent_Length_WithEmptyText_ReturnsZero(t *testing.T) {
	mc, _ := valueobject.NewMessageContent("A")
	mc.Text = ""

	if mc.Length() != 0 {
		t.Errorf("expected 0, got %d", mc.Length())
	}
}