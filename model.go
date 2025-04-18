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
	AuthorID  string
}

type BrokenPopmamaArticleCSC struct {
	OldID      string
	AuthorID   string
	AuthorKey  string
	CreatedBy  string
	CreatorKey string
}

type UnfixedPosts struct {
	ID     int    `json:"id"`
	Reason string `json:"reason"`
	Error  string `json:"error"`
}

type AuthorOS struct {
	UUID    string `json:"uuid"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Key     string `json:"key"`
	Avatar  string `json:"avatar"`
	IsBrand bool   `json:"is_brand"`
}

type AuthorGetResult struct {
	Index  string    `json:"_index,omitempty"`
	ID     string    `json:"_id,omitempty"`
	Found  bool      `json:"found,omitempty"`
	Source *AuthorOS `json:"_source,omitempty"`
}
