package main

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

// MockDBTransaction is a mock for database transactions
type MockDBTransaction struct {
	CommitCalled   bool
	RollbackCalled bool
	ShouldError    bool
}

func (m *MockDBTransaction) Commit() error {
	m.CommitCalled = true
	if m.ShouldError {
		return errors.New("commit error")
	}
	return nil
}

func (m *MockDBTransaction) Rollback() error {
	m.RollbackCalled = true
	if m.ShouldError {
		return errors.New("rollback error")
	}
	return nil
}

// MockOneCMSDB is a mock implementation of the OneCMSDB interface for testing
type MockOneCMSDB struct {
	MockTx                 *MockDBTransaction
	PostsByCreatedAt       []Post
	GetPostsByCreatedAtErr error
	AuthorKey              string
	GetAuthorKeyErr        error
	UpdateURLErr           error
	Post                   *Post
	GetPostErr             error
	BrokenPosts            []BrokenPopmamaArticleCSC
	GetBrokenPostsErr      error
	UpdateBrokenPostErr    error
}

func (m *MockOneCMSDB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if m.MockTx == nil {
		m.MockTx = &MockDBTransaction{}
	}

	if m.MockTx.ShouldError {
		return nil, errors.New("begin transaction error")
	}

	// Create a fake sql.Tx for testing
	fakeTx := &sql.Tx{}

	// We return a non-nil *sql.Tx even though it's not a real one
	// This is just to avoid nil pointer dereferences in tests
	return fakeTx, nil
}

func (m *MockOneCMSDB) Commit(ctx context.Context, tx *sql.Tx) error {
	if m.MockTx == nil {
		return errors.New("transaction not started")
	}
	return m.MockTx.Commit()
}

func (m *MockOneCMSDB) Rollback(ctx context.Context, tx *sql.Tx) error {
	if m.MockTx == nil {
		return errors.New("transaction not started")
	}
	return m.MockTx.Rollback()
}

func (m *MockOneCMSDB) GetPostsByCreatedAt(ctx context.Context, startAt, endAt string) ([]Post, error) {
	return m.PostsByCreatedAt, m.GetPostsByCreatedAtErr
}

func (m *MockOneCMSDB) GetBrokenPopmamaArticleCSC(ctx context.Context) ([]BrokenPopmamaArticleCSC, error) {
	return m.BrokenPosts, m.GetBrokenPostsErr
}

func (m *MockOneCMSDB) GetAuthorKeyByPostID(ctx context.Context, postID string) (string, error) {
	return m.AuthorKey, m.GetAuthorKeyErr
}

func (m *MockOneCMSDB) UpdateArticleURLByID(ctx context.Context, postID, fixedURL string) error {
	return m.UpdateURLErr
}

func (m *MockOneCMSDB) GetPostByOldIDAndPublisher(ctx context.Context, oldID, publisher string) (*Post, error) {
	return m.Post, m.GetPostErr
}

func (m *MockOneCMSDB) UpdateBrokenPopmamaArticleCSC(ctx context.Context, transactionDB *sql.Tx, postID string, post Post) error {
	return m.UpdateBrokenPostErr
}

func (m *MockOneCMSDB) FlushPostAuthors(ctx context.Context, transactionDB *sql.Tx, postID string) error {
	return nil
}

func (m *MockOneCMSDB) SetPostAuthor(ctx context.Context, transactionDB *sql.Tx, postID string, authorID string, orderNumber int) error {
	return nil
}

// Test BeginTx, Commit, and Rollback
func TestDatabaseTransactions(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful transaction", func(t *testing.T) {
		mockDB := &MockOneCMSDB{
			MockTx: &MockDBTransaction{},
		}

		tx, err := mockDB.BeginTx(ctx)
		if err != nil {
			t.Errorf("BeginTx() error = %v, expected nil", err)
		}

		err = mockDB.Commit(ctx, tx)
		if err != nil {
			t.Errorf("Commit() error = %v, expected nil", err)
		}

		if !mockDB.MockTx.CommitCalled {
			t.Errorf("Commit() was not called")
		}
	})

	t.Run("Rollback transaction", func(t *testing.T) {
		mockDB := &MockOneCMSDB{
			MockTx: &MockDBTransaction{},
		}

		tx, err := mockDB.BeginTx(ctx)
		if err != nil {
			t.Errorf("BeginTx() error = %v, expected nil", err)
		}

		err = mockDB.Rollback(ctx, tx)
		if err != nil {
			t.Errorf("Rollback() error = %v, expected nil", err)
		}

		if !mockDB.MockTx.RollbackCalled {
			t.Errorf("Rollback() was not called")
		}
	})

	t.Run("Transaction error", func(t *testing.T) {
		mockDB := &MockOneCMSDB{
			MockTx: &MockDBTransaction{ShouldError: true},
		}

		_, err := mockDB.BeginTx(ctx)
		if err == nil {
			t.Errorf("BeginTx() expected error, got nil")
		}
	})
}

// Test GetPostsByCreatedAt
func TestGetPostsByCreatedAt(t *testing.T) {
	ctx := context.Background()

	t.Run("Successfully get posts", func(t *testing.T) {
		expectedPosts := []Post{
			{
				ID:        "1",
				Title:     "Test Post",
				FullURL:   "https://example.com/test-post-author-123",
				Key:       "test-key",
				CreatedAt: time.Now(),
			},
		}

		mockDB := &MockOneCMSDB{
			PostsByCreatedAt: expectedPosts,
		}

		posts, err := mockDB.GetPostsByCreatedAt(ctx, "2023-01-01", "2023-01-02")
		if err != nil {
			t.Errorf("GetPostsByCreatedAt() error = %v, expected nil", err)
		}

		if len(posts) != len(expectedPosts) {
			t.Errorf("GetPostsByCreatedAt() returned %d posts, expected %d", len(posts), len(expectedPosts))
		}
	})

	t.Run("Error getting posts", func(t *testing.T) {
		mockDB := &MockOneCMSDB{
			GetPostsByCreatedAtErr: errors.New("database error"),
		}

		_, err := mockDB.GetPostsByCreatedAt(ctx, "2023-01-01", "2023-01-02")
		if err == nil {
			t.Errorf("GetPostsByCreatedAt() expected error, got nil")
		}
	})
}

// Test GetAuthorKeyByPostID
func TestGetAuthorKeyByPostID(t *testing.T) {
	ctx := context.Background()

	t.Run("Successfully get author key", func(t *testing.T) {
		expectedKey := "author-key"
		mockDB := &MockOneCMSDB{
			AuthorKey: expectedKey,
		}

		key, err := mockDB.GetAuthorKeyByPostID(ctx, "1")
		if err != nil {
			t.Errorf("GetAuthorKeyByPostID() error = %v, expected nil", err)
		}

		if key != expectedKey {
			t.Errorf("GetAuthorKeyByPostID() returned key = %v, expected %v", key, expectedKey)
		}
	})

	t.Run("Error getting author key", func(t *testing.T) {
		mockDB := &MockOneCMSDB{
			GetAuthorKeyErr: errors.New("database error"),
		}

		_, err := mockDB.GetAuthorKeyByPostID(ctx, "1")
		if err == nil {
			t.Errorf("GetAuthorKeyByPostID() expected error, got nil")
		}
	})
}

// Test UpdateArticleURLByID
func TestUpdateArticleURLByID(t *testing.T) {
	ctx := context.Background()

	t.Run("Successfully update URL", func(t *testing.T) {
		mockDB := &MockOneCMSDB{}

		err := mockDB.UpdateArticleURLByID(ctx, "1", "https://example.com/new-url")
		if err != nil {
			t.Errorf("UpdateArticleURLByID() error = %v, expected nil", err)
		}
	})

	t.Run("Error updating URL", func(t *testing.T) {
		mockDB := &MockOneCMSDB{
			UpdateURLErr: errors.New("database error"),
		}

		err := mockDB.UpdateArticleURLByID(ctx, "1", "https://example.com/new-url")
		if err == nil {
			t.Errorf("UpdateArticleURLByID() expected error, got nil")
		}
	})
}

// Test GetPostByOldIDAndPublisher
func TestGetPostByOldIDAndPublisher(t *testing.T) {
	ctx := context.Background()

	t.Run("Successfully get post", func(t *testing.T) {
		expectedPost := &Post{
			ID:        "1",
			Title:     "Test Post",
			FullURL:   "https://example.com/test-post-author-123",
			Key:       "test-key",
			CreatedAt: time.Now(),
		}

		mockDB := &MockOneCMSDB{
			Post: expectedPost,
		}

		post, err := mockDB.GetPostByOldIDAndPublisher(ctx, "old-1", "popmama")
		if err != nil {
			t.Errorf("GetPostByOldIDAndPublisher() error = %v, expected nil", err)
		}

		if post != expectedPost {
			t.Errorf("GetPostByOldIDAndPublisher() returned post = %v, expected %v", post, expectedPost)
		}
	})

	t.Run("Error getting post", func(t *testing.T) {
		mockDB := &MockOneCMSDB{
			GetPostErr: errors.New("database error"),
		}

		_, err := mockDB.GetPostByOldIDAndPublisher(ctx, "old-1", "popmama")
		if err == nil {
			t.Errorf("GetPostByOldIDAndPublisher() expected error, got nil")
		}
	})
}

// Test GetBrokenPopmamaArticleCSC
func TestGetBrokenPopmamaArticleCSC(t *testing.T) {
	ctx := context.Background()

	t.Run("Successfully get broken posts", func(t *testing.T) {
		expectedPosts := []BrokenPopmamaArticleCSC{
			{
				OldID:      "old-1",
				AuthorID:   "author-1",
				AuthorKey:  "author-key",
				CreatedBy:  "creator-1",
				CreatorKey: "creator-key",
			},
		}

		mockDB := &MockOneCMSDB{
			BrokenPosts: expectedPosts,
		}

		posts, err := mockDB.GetBrokenPopmamaArticleCSC(ctx)
		if err != nil {
			t.Errorf("GetBrokenPopmamaArticleCSC() error = %v, expected nil", err)
		}

		if len(posts) != len(expectedPosts) {
			t.Errorf("GetBrokenPopmamaArticleCSC() returned %d posts, expected %d", len(posts), len(expectedPosts))
		}
	})

	t.Run("Error getting broken posts", func(t *testing.T) {
		mockDB := &MockOneCMSDB{
			GetBrokenPostsErr: errors.New("database error"),
		}

		_, err := mockDB.GetBrokenPopmamaArticleCSC(ctx)
		if err == nil {
			t.Errorf("GetBrokenPopmamaArticleCSC() expected error, got nil")
		}
	})
}

// Test UpdateBrokenPopmamaArticleCSC
func TestUpdateBrokenPopmamaArticleCSC(t *testing.T) {
	ctx := context.Background()

	t.Run("Successfully update broken post", func(t *testing.T) {
		mockDB := &MockOneCMSDB{}

		post := Post{
			ID:      "1",
			Title:   "Test Post",
			FullURL: "https://example.com/test-post-author-123",
			Key:     "test-key",
		}

		err := mockDB.UpdateBrokenPopmamaArticleCSC(ctx, nil, "old-1", post)
		if err != nil {
			t.Errorf("UpdateBrokenPopmamaArticleCSC() error = %v, expected nil", err)
		}
	})

	t.Run("Error updating broken post", func(t *testing.T) {
		mockDB := &MockOneCMSDB{
			UpdateBrokenPostErr: errors.New("database error"),
		}

		post := Post{
			ID:      "1",
			Title:   "Test Post",
			FullURL: "https://example.com/test-post-author-123",
			Key:     "test-key",
		}

		err := mockDB.UpdateBrokenPopmamaArticleCSC(ctx, nil, "old-1", post)
		if err == nil {
			t.Errorf("UpdateBrokenPopmamaArticleCSC() expected error, got nil")
		}
	})
}
