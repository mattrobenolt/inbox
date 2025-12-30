package gmail

import "time"

// Thread represents a conversation thread in the inbox
type Thread struct {
	ThreadID      string    `json:"thread_id"`
	Subject       string    `json:"subject"`
	From          string    `json:"from"`
	Snippet       string    `json:"snippet"`
	Date          time.Time `json:"date"`
	MessageCount  int       `json:"message_count"`
	Unread        bool      `json:"unread"`
	HasAttachment bool      `json:"has_attachment"`
	Labels        []string  `json:"labels"`

	// Account tracking
	AccountIndex int    `json:"account_index"` // Index into clients array
	AccountName  string `json:"account_name"`  // Display name for account

	// Loaded indicates if metadata has been fetched (not serialized)
	Loaded bool `json:"-"`

	// SearchOnly indicates this thread was loaded from a search query (not serialized)
	SearchOnly bool `json:"-"`
}

// Message represents a single email message
type Message struct {
	ID          string       `json:"id"`
	ThreadID    string       `json:"thread_id"`
	From        string       `json:"from"`
	To          string       `json:"to"`
	Cc          string       `json:"cc,omitempty"`
	Subject     string       `json:"subject"`
	Date        time.Time    `json:"date"`
	Snippet     string       `json:"snippet"`
	BodyText    string       `json:"body_text,omitempty"`
	BodyHTML    string       `json:"body_html,omitempty"`
	Labels      []string     `json:"labels"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment represents a file attachment
type Attachment struct {
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	// AttachmentID for fetching the actual data later
	AttachmentID string `json:"attachment_id,omitempty"`
}

// InboxResponse is what we'd return from ListInbox RPC
type InboxResponse struct {
	Threads       []Thread `json:"threads"`
	NextPageToken string   `json:"next_page_token,omitempty"`
}

// Label represents a Gmail label
type Label struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"` // "system" or "user"
	MessagesTotal  int64  `json:"messages_total"`
	MessagesUnread int64  `json:"messages_unread"`
}
