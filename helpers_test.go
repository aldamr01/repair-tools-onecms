package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestPrettyF(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "String map",
			input:    map[string]string{"key": "value"},
			expected: "{\n    \"key\": \"value\"\n}",
		},
		{
			name:     "Empty slice",
			input:    []string{},
			expected: "[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PrettyF(tt.input)
			if result != tt.expected {
				t.Errorf("PrettyF() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEncode(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name:        "Valid struct",
			input:       struct{ Name string }{"Test"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := Encode(tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("Encode() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError {
				var result bytes.Buffer
				json.NewEncoder(&result).Encode(tt.input)
				if !reflect.DeepEqual(buf.Bytes(), result.Bytes()) {
					t.Errorf("Encode() = %v, want %v", buf.String(), result.String())
				}
			}
		})
	}
}

func TestChunk(t *testing.T) {
	tests := []struct {
		name      string
		items     []int
		chunkSize int
		expected  [][]int
	}{
		{
			name:      "Empty list",
			items:     []int{},
			chunkSize: 2,
			expected:  nil,
		},
		{
			name:      "Chunk size 2",
			items:     []int{1, 2, 3, 4, 5},
			chunkSize: 2,
			expected:  [][]int{{1, 2}, {3, 4}, {5}},
		},
		{
			name:      "Chunk size larger than list",
			items:     []int{1, 2, 3},
			chunkSize: 5,
			expected:  [][]int{{1, 2, 3}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Chunk(tt.items, tt.chunkSize)

			if len(tt.items) == 0 && (result == nil || len(result) == 0) {
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Chunk() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseDataAs(t *testing.T) {
	tests := []struct {
		name        string
		source      interface{}
		dest        interface{}
		expectError bool
	}{
		{
			name:        "Valid parsing",
			source:      map[string]interface{}{"name": "John", "age": 30},
			dest:        &struct{ Name string }{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseDataAs(tt.source, tt.dest)
			if (err != nil) != tt.expectError {
				t.Errorf("ParseDataAs() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expected    string
		expectError bool
	}{
		{
			name:        "Simple map",
			input:       map[string]string{"key": "value"},
			expected:    "{\"key\":\"value\"}",
			expectError: false,
		},
		{
			name:        "Simple struct",
			input:       struct{ Name string }{"Test"},
			expected:    "{\"Name\":\"Test\"}",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToString(tt.input)
			if (err != nil) != tt.expectError {
				t.Errorf("ToString() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if result != tt.expected {
				t.Errorf("ToString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFixURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		newAuthorKey  string
		expected      string
		expectError   bool
		expectedError string
	}{
		{
			name:         "Valid URL",
			url:          "https://example.com/article-title-oldkey-12345",
			newAuthorKey: "newkey",
			expected:     "https://example.com/article-title-newkey-12345",
			expectError:  false,
		},
		{
			name:         "URL with multiple hyphens",
			url:          "https://example.com/this-is-an-article-title-with-hyphens-oldkey-12345",
			newAuthorKey: "newkey",
			expected:     "https://example.com/this-is-an-article-title-with-hyphens-newkey-12345",
			expectError:  false,
		},
		{
			name:          "Invalid URL - no segments",
			url:           "",
			newAuthorKey:  "newkey",
			expected:      "",
			expectError:   true,
			expectedError: "invalid URL format",
		},
		{
			name:          "Invalid URL - not enough segments",
			url:           "https://example.com/title",
			newAuthorKey:  "newkey",
			expected:      "",
			expectError:   true,
			expectedError: "invalid URL structure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FixURL(tt.url, tt.newAuthorKey)
			if (err != nil) != tt.expectError {
				t.Errorf("FixURL() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if tt.expectError && err.Error() != tt.expectedError {
				t.Errorf("FixURL() error message = %v, want %v", err.Error(), tt.expectedError)
				return
			}
			if result != tt.expected {
				t.Errorf("FixURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}
