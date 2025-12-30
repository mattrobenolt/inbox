package gmail

import (
	"encoding/base64"
	"slices"
	"time"

	"google.golang.org/api/gmail/v1"
)

// GmailToThread converts Gmail API messages to our Thread type
func GmailToThread(messages []*gmail.Message) *Thread {
	if len(messages) == 0 {
		return nil
	}

	// Use the latest message for most fields
	latest := messages[0]
	for _, msg := range messages {
		if msg.InternalDate > latest.InternalDate {
			latest = msg
		}
	}

	thread := &Thread{
		ThreadID:     latest.ThreadId,
		Snippet:      latest.Snippet,
		Date:         time.UnixMilli(latest.InternalDate),
		MessageCount: len(messages),
		Labels:       latest.LabelIds,
	}

	// Check if any message is unread
	for _, msg := range messages {
		if slices.Contains(msg.LabelIds, "UNREAD") {
			thread.Unread = true
		}
	}

	// Extract headers from latest message
	if latest.Payload != nil {
		for _, header := range latest.Payload.Headers {
			switch header.Name {
			case "Subject":
				thread.Subject = header.Value
			case "From":
				thread.From = header.Value
			}
		}

		// Check for attachments
		thread.HasAttachment = hasAttachments(latest.Payload)
	}

	return thread
}

// GmailToMessage converts a Gmail API message to our Message type
func GmailToMessage(msg *gmail.Message) *Message {
	message := &Message{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Snippet:  msg.Snippet,
		Date:     time.UnixMilli(msg.InternalDate),
		Labels:   msg.LabelIds,
	}

	if msg.Payload != nil {
		// Extract headers
		for _, header := range msg.Payload.Headers {
			switch header.Name {
			case "From":
				message.From = header.Value
			case "To":
				message.To = header.Value
			case "Cc":
				message.Cc = header.Value
			case "Subject":
				message.Subject = header.Value
			}
		}

		// Extract body and attachments
		message.BodyText, message.BodyHTML, message.Attachments = extractBodyAndAttachments(
			msg.Payload,
		)
	}

	return message
}

// hasAttachments checks if a message payload has attachments
func hasAttachments(payload *gmail.MessagePart) bool {
	if payload.Filename != "" && payload.Body != nil && payload.Body.AttachmentId != "" {
		return true
	}

	return slices.ContainsFunc(payload.Parts, hasAttachments)
}

// extractBodyAndAttachments walks the MIME parts and extracts body content and attachments
func extractBodyAndAttachments(
	payload *gmail.MessagePart,
) (text string, html string, attachments []Attachment) {
	// If this part has a filename, it's an attachment
	if payload.Filename != "" && payload.Body != nil {
		if payload.Body.AttachmentId != "" {
			attachments = append(attachments, Attachment{
				Filename:     payload.Filename,
				MimeType:     payload.MimeType,
				Size:         payload.Body.Size,
				AttachmentID: payload.Body.AttachmentId,
			})
		}
		return
	}

	// If this part has body data, extract it
	if payload.Body != nil && payload.Body.Data != "" {
		decoded, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			switch payload.MimeType {
			case "text/plain":
				text = string(decoded)
			case "text/html":
				html = string(decoded)
			}
		}
	}

	// Recursively process parts
	for _, part := range payload.Parts {
		partText, partHTML, partAttachments := extractBodyAndAttachments(part)
		if partText != "" && text == "" {
			text = partText
		}
		if partHTML != "" && html == "" {
			html = partHTML
		}
		attachments = append(attachments, partAttachments...)
	}

	return
}
