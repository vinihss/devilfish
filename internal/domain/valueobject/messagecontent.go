package valueobject

import (
	"errors"
)

// MessageContent represents the content of a message as a value object.
// Value objects are immutable and compared by their values.
type MessageContent struct {
	Text     string // Main text content
	HTML     string // HTML formatted content
	Markdown string // Markdown formatted content
}

// ErrEmptyContent is returned when content is empty.
var ErrEmptyContent = errors.New("message content cannot be empty")

// NewMessageContent creates a new MessageContent value object.
func NewMessageContent(text string) (*MessageContent, error) {
	if text == "" {
		return nil, ErrEmptyContent
	}

	return &MessageContent{
		Text: text,
	}, nil
}

// NewMessageContentWithFormats creates a MessageContent with all formats.
func NewMessageContentWithFormats(text, html, markdown string) (*MessageContent, error) {
	if text == "" {
		return nil, ErrEmptyContent
	}

	return &MessageContent{
		Text:     text,
		HTML:     html,
		Markdown: markdown,
	}, nil
}

// Equals compares two MessageContent values for equality.
func (mc *MessageContent) Equals(other *MessageContent) bool {
	if mc == nil || other == nil {
		return mc == other
	}
	return mc.Text == other.Text &&
		mc.HTML == other.HTML &&
		mc.Markdown == other.Markdown
}

// HasHTML returns true if HTML content is present.
func (mc *MessageContent) HasHTML() bool {
	return mc.HTML != ""
}

// HasMarkdown returns true if Markdown content is present.
func (mc *MessageContent) HasMarkdown() bool {
	return mc.Markdown != ""
}

// Length returns the length of the text content.
func (mc *MessageContent) Length() int {
	return len(mc.Text)
}
