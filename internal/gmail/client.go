package gmail

import (
	"context"
	"encoding/base64"
	"io"
	"strings"

	"google.golang.org/api/gmail/v1"
)

// Client wraps Gmail API service
type Client struct {
	srv *gmail.Service
}

// NewClient creates a new Gmail client
func NewClient(srv *gmail.Service) *Client {
	return &Client{srv: srv}
}

// ListInbox fetches inbox thread IDs (without metadata for efficiency)
func (c *Client) ListInbox(
	ctx context.Context,
	limit int64,
	pageToken string,
) (*InboxResponse, error) {
	// List threads in inbox (just IDs)
	req := c.srv.Users.Threads.List("me").
		LabelIds("INBOX").
		MaxResults(limit)

	if pageToken != "" {
		req = req.PageToken(pageToken)
	}

	res, err := req.Do()
	if err != nil {
		return nil, err
	}

	// Return thread stubs with just IDs (metadata not loaded)
	threads := make([]Thread, 0, len(res.Threads))
	for _, threadRef := range res.Threads {
		threads = append(threads, Thread{
			ThreadID: threadRef.Id,
			Loaded:   false,
		})
	}

	return &InboxResponse{
		Threads:       threads,
		NextPageToken: res.NextPageToken,
	}, nil
}

// SearchInbox fetches thread IDs that match the Gmail search query.
func (c *Client) SearchInbox(
	ctx context.Context,
	query string,
	limit int64,
	pageToken string,
) (*InboxResponse, error) {
	req := c.srv.Users.Threads.List("me").
		LabelIds("INBOX").
		Q(query).
		MaxResults(limit)

	if pageToken != "" {
		req = req.PageToken(pageToken)
	}

	res, err := req.Do()
	if err != nil {
		return nil, err
	}

	threads := make([]Thread, 0, len(res.Threads))
	for _, threadRef := range res.Threads {
		threads = append(threads, Thread{
			ThreadID: threadRef.Id,
			Loaded:   false,
		})
	}

	return &InboxResponse{
		Threads:       threads,
		NextPageToken: res.NextPageToken,
	}, nil
}

// GetThreadMetadata fetches metadata for a single thread (for lazy loading)
func (c *Client) GetThreadMetadata(ctx context.Context, threadID string) (*Thread, error) {
	// Fetch with metadata format (headers + snippet, no body)
	thread, err := c.srv.Users.Threads.Get("me", threadID).Format("metadata").Do()
	if err != nil {
		return nil, err
	}

	// Convert to our Thread type
	t := GmailToThread(thread.Messages)
	if t != nil {
		t.Loaded = true
	}
	return t, nil
}

// GetThread fetches all messages in a thread
func (c *Client) GetThread(ctx context.Context, threadID string) ([]Message, error) {
	thread, err := c.srv.Users.Threads.Get("me", threadID).Format("full").Do()
	if err != nil {
		return nil, err
	}

	messages := make([]Message, 0, len(thread.Messages))
	for _, msg := range thread.Messages {
		messages = append(messages, *GmailToMessage(msg))
	}

	return messages, nil
}

// GetMessage fetches a single message with full body
func (c *Client) GetMessage(ctx context.Context, messageID string) (*Message, error) {
	msg, err := c.srv.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return nil, err
	}

	return GmailToMessage(msg), nil
}

// GetLabels fetches all labels
func (c *Client) GetLabels(ctx context.Context) ([]Label, error) {
	res, err := c.srv.Users.Labels.List("me").Do()
	if err != nil {
		return nil, err
	}

	labels := make([]Label, 0, len(res.Labels))
	for _, l := range res.Labels {
		labels = append(labels, Label{
			ID:             l.Id,
			Name:           l.Name,
			Type:           l.Type,
			MessagesTotal:  l.MessagesTotal,
			MessagesUnread: l.MessagesUnread,
		})
	}

	return labels, nil
}

// MarkThreadRead marks a thread as read by removing the UNREAD label
func (c *Client) MarkThreadRead(ctx context.Context, threadID string) error {
	req := &gmail.ModifyThreadRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}
	_, err := c.srv.Users.Threads.Modify("me", threadID, req).Context(ctx).Do()
	return err
}

// MarkThreadUnread marks a thread as unread by adding the UNREAD label
func (c *Client) MarkThreadUnread(ctx context.Context, threadID string) error {
	req := &gmail.ModifyThreadRequest{
		AddLabelIds: []string{"UNREAD"},
	}
	_, err := c.srv.Users.Threads.Modify("me", threadID, req).Context(ctx).Do()
	return err
}

// DownloadAttachment downloads an attachment and returns the raw bytes
func (c *Client) DownloadAttachment(
	ctx context.Context,
	messageID, attachmentID string,
) ([]byte, error) {
	attachment, err := c.srv.Users.Messages.Attachments.Get("me", messageID, attachmentID).
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}

	// Decode the base64 data
	data, err := base64.URLEncoding.DecodeString(attachment.Data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// DownloadAttachmentToWriter downloads an attachment and writes it to the provided writer
// This streams the decode to avoid double-buffering in memory
func (c *Client) DownloadAttachmentToWriter(
	ctx context.Context,
	messageID, attachmentID string,
	w io.Writer,
) error {
	attachment, err := c.srv.Users.Messages.Attachments.Get("me", messageID, attachmentID).
		Context(ctx).
		Do()
	if err != nil {
		return err
	}

	// Stream decode the base64 data directly to the writer
	decoder := base64.NewDecoder(base64.URLEncoding, strings.NewReader(attachment.Data))
	_, err = io.Copy(w, decoder)
	return err
}

// GetAttachmentData returns the raw base64url-encoded attachment data
// This is useful for passing to the image transformer which expects base64url format
func (c *Client) GetAttachmentData(
	ctx context.Context,
	messageID, attachmentID string,
) (string, error) {
	attachment, err := c.srv.Users.Messages.Attachments.Get("me", messageID, attachmentID).
		Context(ctx).
		Do()
	if err != nil {
		return "", err
	}
	return attachment.Data, nil
}
