package image

import (
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"  // Register GIF format
	_ "image/jpeg" // Register JPEG format
	"image/png"
	"io"
	"strings"
)

// ImageTransformer transforms image data for display in terminals
// It implements io.Reader and handles format conversion:
// - PNG: passes through with base64url -> base64 translation
// - JPEG/GIF/etc: decodes -> encodes to PNG -> base64
type ImageTransformer struct {
	source      *strings.Reader
	contentType string
	// For non-PNG: converted base64 PNG data
	converted   io.Reader
	initialized bool
	// For cleanup: if converted is a PipeReader, we need to close it
	pipeReader *io.PipeReader
}

// NewImageTransformer creates a new image transformer
// base64urlData is the base64url-encoded image data from Gmail
// contentType is the MIME type (e.g., "image/png", "image/jpeg")
// Returns the transformer and the image dimensions (width, height)
func NewImageTransformer(
	base64urlData string,
	contentType string,
) (*ImageTransformer, int, int, error) {
	t := &ImageTransformer{
		source:      strings.NewReader(base64urlData),
		contentType: contentType,
	}

	// Decode config to get dimensions without decoding full image
	width, height, err := t.getDimensions()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("getting image dimensions: %w", err)
	}

	return t, width, height, nil
}

// getDimensions decodes just the image config to get dimensions
func (t *ImageTransformer) getDimensions() (int, int, error) {
	// Reset to start
	t.source.Seek(0, io.SeekStart)

	// Create streaming pipeline for decoding
	translator := &charTranslator{r: t.source}
	decoder := base64.NewDecoder(base64.StdEncoding, translator)

	// Decode just the config (fast - only reads headers)
	config, _, err := image.DecodeConfig(decoder)
	if err != nil {
		return 0, 0, err
	}

	return config.Width, config.Height, nil
}

// Read implements io.Reader
func (t *ImageTransformer) Read(p []byte) (n int, err error) {
	// Convert to PNG on first read (lazy init)
	if !t.initialized {
		if err := t.convertToPNG(); err != nil {
			return 0, err
		}
		t.initialized = true
	}

	// Read from converted data
	return t.converted.Read(p)
}

// convertToPNG converts non-PNG images to PNG format
func (t *ImageTransformer) convertToPNG() error {
	// Reset source to beginning
	t.source.Seek(0, io.SeekStart)

	// Create streaming pipeline:
	// strings.Reader → charTranslator → base64.Decoder → image.Decode
	translator := &charTranslator{r: t.source}
	decoder := base64.NewDecoder(base64.StdEncoding, translator)

	// Decode image directly from the streaming decoder (format auto-detected)
	img, _, err := image.Decode(decoder)
	if err != nil {
		return fmt.Errorf("decoding %s image: %w", t.contentType, err)
	}

	// Create pipe for streaming PNG encode → base64 encode
	pr, pw := io.Pipe()

	// Start goroutine to encode PNG → base64 → pipe
	go func() {
		// Wrap pipe writer with base64 encoder
		encoder := base64.NewEncoder(base64.StdEncoding, pw)

		// Encode image as PNG, writing to base64 encoder
		err := png.Encode(encoder, img)

		// Close encoder to flush any buffered data
		encoder.Close()

		// Close pipe writer (with error if encoding failed)
		pw.CloseWithError(err)
	}()

	// Store pipe reader - reads will stream from the goroutine
	t.converted = pr
	t.pipeReader = pr // Save for cleanup

	return nil
}

// Close cleans up resources, particularly stopping the background goroutine
// for non-PNG image conversions. It's safe to call multiple times.
func (t *ImageTransformer) Close() error {
	if t.pipeReader != nil {
		// Closing the pipe reader will cause the writer to error,
		// allowing the goroutine to exit
		err := t.pipeReader.Close()
		t.pipeReader = nil
		return err
	}
	return nil
}

// charTranslator translates base64url to standard base64 on-the-fly
type charTranslator struct {
	r io.Reader
}

func (c *charTranslator) Read(p []byte) (n int, err error) {
	n, err = c.r.Read(p)
	// Translate characters in-place: - to +, _ to /
	for i := 0; i < n; i++ {
		switch p[i] {
		case '-':
			p[i] = '+'
		case '_':
			p[i] = '/'
		}
	}
	return n, err
}
