package main

import (
	"time"
)

type Post struct {
	ID        string
	Title     string
	Key       string
	FullURL   string
	CreatedBy string
	CreatedAt time.Time
}

type unfixedPosts struct {
	ID     int    `json:"id"`
	Reason string `json:"reason"`
	Error  string `json:"error"`
}
