package service

import (
	"os"
	"testing"
)

func TestProcessImage(t *testing.T) {
	// Example usage
	inputFile := "./test_in.png"   // Update this path
	outputFile := "./test_out.jpg" // Update this path
	text := "Cool beans! Please clean them up thoroughly unless you want to PAYYYYY me"

	// Read the input image file
	imageBytes, err := os.Open(inputFile)
	if err != nil {
		t.Error(err)
	}

	// Process the image and get the result
	resultBytes, err := ProcessImage(imageBytes, text)
	if err != nil {
		t.Error(err)
	}

	// Write the result to a file
	if err := os.WriteFile(outputFile, resultBytes.Bytes(), 0644); err != nil {
		t.Error(err)
	}
}
