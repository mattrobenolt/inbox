package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/term"

	"go.withmatt.com/inbox/internal/image"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: imgtest <image-path>")
		fmt.Println("  Supported formats: .jpg, .jpeg, .png, .gif")
		os.Exit(1)
	}

	imagePath := os.Args[1]
	fmt.Printf("Testing image: %s\n\n", imagePath)

	// Determine content type from extension
	contentType := "image/png"
	ext := filepath.Ext(imagePath)
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".png":
		contentType = "image/png"
	default:
		fmt.Printf("Unsupported extension: %s\n", ext)
		os.Exit(1)
	}

	// Read file and convert to base64url (simulating Gmail API format)
	data, err := os.ReadFile(imagePath)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Simulate Gmail's base64url encoding
	base64urlData := encodeBase64URL(data)

	fmt.Printf("Content-Type: %s\n", contentType)
	fmt.Printf("File size: %d bytes\n", len(data))
	fmt.Printf("Base64url size: %d bytes\n\n", len(base64urlData))

	// Get terminal size
	termWidth, termHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Default to reasonable size if we can't detect
		termWidth = 80
		termHeight = 24
	}

	// Create transformer and get dimensions
	transformer, imgWidth, imgHeight, err := image.NewImageTransformer(base64urlData, contentType)
	if err != nil {
		fmt.Printf("Error creating transformer: %v\n", err)
		os.Exit(1)
	}
	defer transformer.Close()

	fmt.Printf(
		"Image dimensions: %dx%d (aspect: %.2f)\n",
		imgWidth,
		imgHeight,
		float64(imgWidth)/float64(imgHeight),
	)
	fmt.Printf("Terminal size: %dx%d\n\n", termWidth, termHeight)

	// Test 1: Full screen centered
	maxCols := termWidth - 4  // Leave some margin
	maxRows := termHeight - 6 // Leave room for text

	// Calculate aspect ratios
	termAspect := float64(maxCols) / float64(maxRows)
	imgAspect := float64(imgWidth) / float64(imgHeight)

	var constraintParam string
	var resultCols, resultRows int

	if imgAspect > termAspect {
		// Landscape: constrain by width, calculate resulting height
		resultCols = maxCols
		resultRows = int(float64(maxCols) * float64(imgHeight) / float64(imgWidth))
		constraintParam = fmt.Sprintf("c=%d", maxCols)
		fmt.Printf("1. Full screen CENTERED (landscape):\n")
		fmt.Printf(
			"   Constraining width to %d cols → height will be ~%d rows\n",
			resultCols,
			resultRows,
		)
	} else {
		// Portrait: constrain by height, calculate resulting width
		resultRows = maxRows
		resultCols = int(float64(maxRows) * float64(imgWidth) / float64(imgHeight))
		constraintParam = fmt.Sprintf("r=%d", maxRows)
		fmt.Printf("1. Full screen CENTERED (portrait):\n")
		fmt.Printf("   Constraining height to %d rows → width will be ~%d cols\n", resultRows, resultCols)
	}

	// Calculate centering offsets
	offsetX := (termWidth - resultCols) / 2
	offsetY := (termHeight - resultRows) / 2

	fmt.Printf("   Centering at X=%d, Y=%d\n", offsetX, offsetY)

	transformer, _, _, _ = image.NewImageTransformer(base64urlData, contentType)
	defer transformer.Close()
	fmt.Printf("\x1b_Gf=100,a=T,%s,X=%d,Y=%d;", constraintParam, offsetX, offsetY)
	io.Copy(os.Stdout, transformer)
	fmt.Print("\x1b\\")
	fmt.Println()
	time.Sleep(2 * time.Second)

	// Test 2: Medium size
	fmt.Println("2. Medium size (60 columns):")
	transformer, _, _, _ = image.NewImageTransformer(base64urlData, contentType)
	defer transformer.Close()
	fmt.Print("\x1b_Gf=100,a=T,c=60;")
	io.Copy(os.Stdout, transformer)
	fmt.Print("\x1b\\")
	fmt.Println()
	time.Sleep(1 * time.Second)

	// Test 3: Small thumbnail
	fmt.Println("3. Small thumbnail (30 columns):")
	transformer, _, _, _ = image.NewImageTransformer(base64urlData, contentType)
	defer transformer.Close()
	fmt.Print("\x1b_Gf=100,a=T,c=30;")
	io.Copy(os.Stdout, transformer)
	fmt.Print("\x1b\\")
	fmt.Println()

	fmt.Println("=== Test Complete ===")
}

// encodeBase64URL encodes bytes to base64url format (like Gmail API returns)
func encodeBase64URL(data []byte) string {
	// Encode to standard base64
	std := base64.StdEncoding.EncodeToString(data)

	// Translate to URL encoding: + to -, / to _
	result := make([]byte, len(std))
	for i := 0; i < len(std); i++ {
		switch std[i] {
		case '+':
			result[i] = '-'
		case '/':
			result[i] = '_'
		default:
			result[i] = std[i]
		}
	}

	return string(result)
}
