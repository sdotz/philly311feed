package service

import (
	"bytes"
	"io"
	"log"
	"os"

	"github.com/fogleman/gg"
)

const (
	FONT_SIZE     = 42
	CANVAS_WIDTH  = 1179
	CANVAS_HEIGHT = 2200
)

// processImage takes the input image bytes and a text string, fits the image within the canvas, and renders the text.
func ProcessImage(reader io.Reader, text string) (*bytes.Buffer, error) {
	imgBytes, err := io.ReadAll(reader)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile("/tmp/in", imgBytes, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// Load the image
	img, err := gg.LoadImage("/tmp/in")
	if err != nil {
		log.Fatal(err)
	}

	// Create a new context based on the image dimensions
	imgCtx := gg.NewContextForImage(img)

	dc := gg.NewContext(CANVAS_WIDTH, CANVAS_HEIGHT)

	// Calculate the scaling factors for width and height
	widthScale := float64(dc.Width()) / float64(imgCtx.Width())
	heightScale := float64(dc.Height()) / float64(imgCtx.Height())

	// Use the smaller scaling factor to ensure the image fits within the canvas
	scale := widthScale
	if heightScale < widthScale {
		scale = heightScale
	}

	// Calculate the new dimensions
	newWidth := int(float64(imgCtx.Width()) * scale)
	newHeight := int(float64(imgCtx.Height()) * scale)

	imgCtx.Scale(float64(newWidth), float64(newHeight))

	dc.DrawImage(imgCtx.Image(), (dc.Width()-imgCtx.Width())/2, (dc.Height()-imgCtx.Height())/2)

	// Set up the font
	fontPath := "./SF-Pro.ttf"
	if os.Getenv("FUNCTION_TARGET") != "" {
		fontPath = "./serverless_function_source_code/SF-Pro.ttf"
	}

	if err := dc.LoadFontFace(fontPath, FONT_SIZE); err != nil {
		log.Fatal(err)
	}

	// Set the text color
	dc.SetRGB(1, 1, 1) // White

	// Draw the wrapped text
	dc.DrawStringWrapped(text, 20, float64((dc.Height()/2)+(imgCtx.Height()/2))+20, 0, 0, float64(CANVAS_WIDTH-30), 1.5, gg.AlignLeft)

	// Save the resulting image
	if err := dc.SaveJPG("/tmp/out.jpg", 80); err != nil {
		log.Fatal(err)
	}

	f, err := os.ReadFile("/tmp/out.jpg")
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(f), nil
}
