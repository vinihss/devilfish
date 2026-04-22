package entity

import (
	"errors"
	"time"

	"devilfish/internal/domain/valueobject"
)

// MessageType represents the type of message.
type MessageType string

const (
	// MessageTypeText represents a text message.
	MessageTypeText MessageType = "text"
	// MessageTypeImage represents an image message.
	MessageTypeImage MessageType = "image"
	// MessageTypeAudio represents an audio message.
	MessageTypeAudio MessageType = "audio"
	// MessageTypeVideo represents a video message.
	MessageTypeVideo MessageType = "video"
	// MessageTypeDocument represents a document message.
	MessageTypeDocument MessageType = "document"
)

// ErrInvalidMessageType is returned when the message type is invalid.
var ErrInvalidMessageType = errors.New("invalid message type")

// Message represents a message entity in the domain.
type Message struct {
	ID        string                `json:"id"`                 // Unique identifier
	Content   string                `json:"content"`            // Message content
	UserID    string                `json:"user_id"`            // User who sent the message
	Channel   string                `json:"channel"`            // Channel where message was received
	Type      MessageType           `json:"type"`               // Type of message
	Timestamp time.Time             `json:"timestamp"`          // When message was created
	Metadata  map[string]any        `json:"metadata"`           // Additional metadata
	Provider  *valueobject.Provider `json:"provider,omitempty"` // AI provider used
}

// NewMessage creates a new Message entity.
func NewMessage(id, content, userID, channel string, msgType MessageType) (*Message, error) {
	if id == "" {
		return nil, errors.New("message id is required")
	}
	if content == "" {
		return nil, errors.New("message content is required")
	}
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	if channel == "" {
		return nil, errors.New("channel is required")
	}

	if !isValidMessageType(msgType) {
		return nil, ErrInvalidMessageType
	}

	return &Message{
		ID:        id,
		Content:   content,
		UserID:    userID,
		Channel:   channel,
		Type:      msgType,
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}, nil
}

// isValidMessageType checks if the message type is valid.
func isValidMessageType(t MessageType) bool {
	switch t {
	case MessageTypeText, MessageTypeImage, MessageTypeAudio,
		MessageTypeVideo, MessageTypeDocument:
		return true
	}
	return false
}

// SetMetadata sets a metadata value for the message.
func (m *Message) SetMetadata(key string, value any) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]any)
	}
	m.Metadata[key] = value
}

// GetMetadata retrieves a metadata value from the message.
func (m *Message) GetMetadata(key string) (any, bool) {
	if m.Metadata == nil {
		return nil, false
	}
	val, ok := m.Metadata[key]
	return val, ok
}
