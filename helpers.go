package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func PrettyPrint(v interface{}) {
	prettyJSON, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate pretty JSON: %s\n", err)
		return
	}
	fmt.Println(string(prettyJSON))
}

func PrettyF(v interface{}) string {
	prettyJSON, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate pretty JSON: %s\n", err)
		return ""
	}
	return string(prettyJSON)
}

func Encode(data interface{}) (bytes.Buffer, error) {
	// Encode the struct to JSON and write it to a buffer
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return buf, err
	}

	return buf, nil
}

func Chunk[T any](items []T, chunkSize int) [][]T {
	// Return empty slice for empty input
	if len(items) == 0 {
		return [][]T{}
	}

	// Handle zero or negative chunk size
	if chunkSize <= 0 {
		chunkSize = 1 // Set a safe default
	}

	var chunks [][]T
	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}

func ParseDataAs(source, dest interface{}) error {
	dataBytes, err := json.Marshal(source)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	err = json.Unmarshal(dataBytes, dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %v", err)
	}

	return nil
}

func ToString(source interface{}) (string, error) {
	jsonString, err := json.Marshal(source)
	if err != nil {
		return "", err
	}

	return string(jsonString), nil
}

func FixURL(url, newAuthorKey string) (string, error) {
	// Check for empty URL
	if url == "" {
		return "", fmt.Errorf("invalid URL format")
	}

	// Find the last segment after the last "/"
	parts := strings.Split(url, "/")
	if len(parts) <= 1 {
		return "", fmt.Errorf("invalid URL format")
	}

	// Extract the last part, which contains the slug and keys
	lastSegment := parts[len(parts)-1]

	// Split by "-" to extract components
	slugParts := strings.Split(lastSegment, "-")
	if len(slugParts) < 3 {
		return "", fmt.Errorf("invalid URL structure")
	}

	// Replace the Author Key with the new one
	slugParts[len(slugParts)-2] = newAuthorKey

	// Reconstruct the URL
	newLastSegment := strings.Join(slugParts, "-")
	parts[len(parts)-1] = newLastSegment

	// Join everything back into a valid URL
	newURL := strings.Join(parts, "/")

	return newURL, nil
}

func LogError(err error) {
	if err == nil {
		return
	}

	// Define the log file path
	logFile := "error.log"

	// Open file in append mode, create if not exists
	file, fileErr := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if fileErr != nil {
		fmt.Println("Failed to open log file:", fileErr)
		return
	}
	defer file.Close()

	// Format error with timestamp
	logMessage := fmt.Sprintf("%s: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)

	// Write error message to the file
	if _, writeErr := file.WriteString(logMessage); writeErr != nil {
		fmt.Println("Failed to write to log file:", writeErr)
	}
}
