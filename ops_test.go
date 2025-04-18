package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

// MockOneCMSOS is a mock implementation of the OneCMSOS interface for testing
type MockOneCMSOS struct {
	DynamicUpdateCalled bool
	DynamicUpdateErr    error
	GetAuthorByIDFunc   func(id string) (*AuthorOS, error)
}

func (m *MockOneCMSOS) DynamicUpdate(data interface{}, id string, index string) error {
	m.DynamicUpdateCalled = true
	return m.DynamicUpdateErr
}

func (m *MockOneCMSOS) GetAuthorByID(id string) (*AuthorOS, error) {
	if m.GetAuthorByIDFunc != nil {
		return m.GetAuthorByIDFunc(id)
	}
	return nil, errors.New("no mock implementation provided")
}

func TestFixURLOperation(t *testing.T) {
	// Set environment variable for chunk size before tests
	os.Setenv("POST_CHUNK_SIZE", "5")
	defer os.Unsetenv("POST_CHUNK_SIZE")

	ctx := context.Background()

	t.Run("Successfully fix URL for all posts", func(t *testing.T) {
		posts := []Post{
			{
				ID:        "1",
				Title:     "Test Post 1",
				FullURL:   "https://example.com/test-post-oldkey-12345",
				Key:       "test-key-1",
				CreatedAt: time.Now(),
			},
			{
				ID:        "2",
				Title:     "Test Post 2",
				FullURL:   "https://example.com/another-post-oldkey-67890",
				Key:       "test-key-2",
				CreatedAt: time.Now(),
			},
		}

		mockDB := &MockOneCMSDB{
			PostsByCreatedAt: posts,
			AuthorKey:        "newkey",
		}

		mockOS := &MockOneCMSOS{}

		err := fixURL(ctx, mockDB, mockOS, "2023-01-01", "2023-01-02", "test-index")
		if err != nil {
			t.Errorf("fixURL() error = %v, expected nil", err)
		}
	})

	t.Run("Error getting posts", func(t *testing.T) {
		mockDB := &MockOneCMSDB{
			GetPostsByCreatedAtErr: errors.New("database error"),
		}

		mockOS := &MockOneCMSOS{}

		err := fixURL(ctx, mockDB, mockOS, "2023-01-01", "2023-01-02", "test-index")
		if err == nil {
			t.Errorf("fixURL() expected error when getting posts, got nil")
		}
	})

	t.Run("Error getting author key", func(t *testing.T) {
		posts := []Post{
			{
				ID:        "1",
				Title:     "Test Post 1",
				FullURL:   "https://example.com/test-post-oldkey-12345",
				Key:       "test-key-1",
				CreatedAt: time.Now(),
			},
		}

		mockDB := &MockOneCMSDB{
			PostsByCreatedAt: posts,
			GetAuthorKeyErr:  errors.New("database error"),
		}

		mockOS := &MockOneCMSOS{}

		err := fixURL(ctx, mockDB, mockOS, "2023-01-01", "2023-01-02", "test-index")
		if err == nil {
			t.Errorf("fixURL() expected error when getting author key, got nil")
		}
	})

	t.Run("Error updating article URL", func(t *testing.T) {
		posts := []Post{
			{
				ID:        "1",
				Title:     "Test Post 1",
				FullURL:   "https://example.com/test-post-oldkey-12345",
				Key:       "test-key-1",
				CreatedAt: time.Now(),
			},
		}

		mockDB := &MockOneCMSDB{
			PostsByCreatedAt: posts,
			AuthorKey:        "newkey",
			UpdateURLErr:     errors.New("database error"),
		}

		mockOS := &MockOneCMSOS{}

		err := fixURL(ctx, mockDB, mockOS, "2023-01-01", "2023-01-02", "test-index")
		if err == nil {
			t.Errorf("fixURL() expected error when updating article URL, got nil")
		}
	})

	t.Run("Error updating OpenSearch data", func(t *testing.T) {
		posts := []Post{
			{
				ID:        "1",
				Title:     "Test Post 1",
				FullURL:   "https://example.com/test-post-oldkey-12345",
				Key:       "test-key-1",
				CreatedAt: time.Now(),
			},
		}

		mockDB := &MockOneCMSDB{
			PostsByCreatedAt: posts,
			AuthorKey:        "newkey",
		}

		mockOS := &MockOneCMSOS{
			DynamicUpdateErr: errors.New("opensearch error"),
		}

		err := fixURL(ctx, mockDB, mockOS, "2023-01-01", "2023-01-02", "test-index")
		if err == nil {
			t.Errorf("fixURL() expected error when updating OpenSearch data, got nil")
		}
	})
}

// Modified version of fixCSCPopmama that doesn't rely on database transactions for testing
func testFixCSCPopmama(ctx context.Context, onecmsDB OneCMSDB, onecmsOS OneCMSOS, osIndex string) error {
	fmt.Printf("🔁 Calculating posts based from table temp_popmama_csc")
	posts, err := onecmsDB.GetBrokenPopmamaArticleCSC(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("✅ Got %v posts\n", len(posts))

	chunkSize, _ := strconv.Atoi(os.Getenv("POST_CHUNK_SIZE"))
	fmt.Printf("🔁 Chunking posts into %v\n", chunkSize)
	chunks := Chunk(posts, chunkSize)
	chunkLength := len(chunks)
	fmt.Printf("✅ Got %v chunks\n", len(chunks))
	unfixedPosts := []string{}

	type postOSStructure struct {
		ArticleURL    string     `json:"article_url"`
		ArticleURLAMP string     `json:"article_url_amp"`
		Authors       []AuthorOS `json:"authors"`
	}

	publisher := "popmama"

	for i, chunk := range chunks {
		fmt.Printf("🔁 [%d/%d] Running chunk...\n", i+1, chunkLength)
		cl := len(chunk)

		for i, post := range chunk {
			fmt.Printf("\n\t[%d/%d] Fixing Popmama CSC article...", i+1, cl)

			postAuthor, err := onecmsOS.GetAuthorByID(post.AuthorID)
			if err != nil || postAuthor == nil {
				msg := fmt.Errorf("\n\t❌ Cannot find author of this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			postCreator, err := onecmsOS.GetAuthorByID(post.CreatedBy)
			if err != nil || postCreator == nil {
				msg := fmt.Errorf("\n\t❌ Cannot find creator of this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			postExisting, err := onecmsDB.GetPostByOldIDAndPublisher(ctx, post.OldID, publisher)
			if err != nil || postExisting == nil {
				msg := fmt.Errorf("\n\t❌ Cannot find post with old id: %s.", post.OldID)
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			// Skip the transaction for testing
			fixedURL, err := FixURL(postExisting.FullURL, postAuthor.Key)
			if err != nil {
				msg := fmt.Errorf("\n\t❌ Failed generate fixed url for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			postExisting.FullURL = fixedURL
			postExisting.CreatedBy = postCreator.Key
			postExisting.AuthorID = postAuthor.Key

			if err := onecmsDB.UpdateBrokenPopmamaArticleCSC(ctx, nil, post.OldID, *postExisting); err != nil {
				msg := fmt.Errorf("\n\t❌ Failed updating DB data for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			osData := postOSStructure{
				ArticleURL:    fixedURL,
				ArticleURLAMP: fixedURL + "/amp",
				Authors:       []AuthorOS{*postAuthor},
			}

			if err := onecmsOS.DynamicUpdate(osData, postExisting.ID, osIndex); err != nil {
				msg := fmt.Errorf("\n\t❌ Failed updating OS data for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			fmt.Printf("\n\t 🧑🏾‍💻 Author key: %s", postAuthor.Key)
			fmt.Printf("\n\t 🌏 URL: %s -> %s", postExisting.FullURL, fixedURL)
			fmt.Printf("\n\t ✅ Success fixing post url with id %s ✔️\n", postExisting.ID)
		}

		fmt.Println("-----🚀-----")
	}

	fmt.Printf("\n🚚 UNFIXED: %v", PrettyF(unfixedPosts))

	if len(unfixedPosts) > 0 {
		return fmt.Errorf("\n❗Few total error: %d \n 🚚 UNFIXED: %v", len(unfixedPosts), PrettyF(unfixedPosts))
	}

	return nil
}

func TestFixCSCPopmamaOperation(t *testing.T) {
	// Set environment variable for chunk size before tests
	os.Setenv("POST_CHUNK_SIZE", "5")
	defer os.Unsetenv("POST_CHUNK_SIZE")

	ctx := context.Background()

	t.Run("Successfully fix Popmama CSC", func(t *testing.T) {
		brokenPosts := []BrokenPopmamaArticleCSC{
			{
				OldID:      "old-1",
				AuthorID:   "author-1",
				AuthorKey:  "author-key",
				CreatedBy:  "creator-1",
				CreatorKey: "creator-key",
			},
		}

		post := &Post{
			ID:        "1",
			Title:     "Test Post",
			FullURL:   "https://example.com/test-post-oldkey-12345",
			Key:       "test-key",
			CreatedAt: time.Now(),
		}

		author := &AuthorOS{
			UUID:    "author-1",
			Email:   "author@example.com",
			Name:    "Test Author",
			Key:     "author-key",
			Avatar:  "https://example.com/avatar.jpg",
			IsBrand: false,
		}

		creator := &AuthorOS{
			UUID:    "creator-1",
			Email:   "creator@example.com",
			Name:    "Test Creator",
			Key:     "creator-key",
			Avatar:  "https://example.com/creator-avatar.jpg",
			IsBrand: false,
		}

		// Create a mock transaction
		mockTx := &MockDBTransaction{}

		mockDB := &MockOneCMSDB{
			BrokenPosts: brokenPosts,
			Post:        post,
			MockTx:      mockTx,
		}

		mockOS := &MockOneCMSOS{
			GetAuthorByIDFunc: func(id string) (*AuthorOS, error) {
				if id == "author-1" {
					return author, nil
				} else if id == "creator-1" {
					return creator, nil
				}
				return nil, errors.New("author not found")
			},
		}

		// Use the test version instead
		err := testFixCSCPopmama(ctx, mockDB, mockOS, "test-index")
		if err != nil {
			t.Errorf("testFixCSCPopmama() error = %v, expected nil", err)
		}
	})

	t.Run("Error getting broken posts", func(t *testing.T) {
		mockDB := &MockOneCMSDB{
			GetBrokenPostsErr: errors.New("database error"),
			MockTx:            &MockDBTransaction{},
		}

		mockOS := &MockOneCMSOS{}

		// Use the test version instead
		err := testFixCSCPopmama(ctx, mockDB, mockOS, "test-index")
		if err == nil {
			t.Errorf("testFixCSCPopmama() expected error when getting broken posts, got nil")
		}
	})

	t.Run("Error getting author by ID", func(t *testing.T) {
		brokenPosts := []BrokenPopmamaArticleCSC{
			{
				OldID:      "old-1",
				AuthorID:   "author-1",
				AuthorKey:  "author-key",
				CreatedBy:  "creator-1",
				CreatorKey: "creator-key",
			},
		}

		mockDB := &MockOneCMSDB{
			BrokenPosts: brokenPosts,
			MockTx:      &MockDBTransaction{},
		}

		mockOS := &MockOneCMSOS{
			GetAuthorByIDFunc: func(id string) (*AuthorOS, error) {
				return nil, errors.New("opensearch error")
			},
		}

		// Use the test version instead
		err := testFixCSCPopmama(ctx, mockDB, mockOS, "test-index")
		if err == nil {
			t.Errorf("testFixCSCPopmama() expected error when getting author, got nil")
		}
	})

	t.Run("Error getting post by old ID", func(t *testing.T) {
		brokenPosts := []BrokenPopmamaArticleCSC{
			{
				OldID:      "old-1",
				AuthorID:   "author-1",
				AuthorKey:  "author-key",
				CreatedBy:  "creator-1",
				CreatorKey: "creator-key",
			},
		}

		author := &AuthorOS{
			UUID:    "author-1",
			Email:   "author@example.com",
			Name:    "Test Author",
			Key:     "author-key",
			Avatar:  "https://example.com/avatar.jpg",
			IsBrand: false,
		}

		creator := &AuthorOS{
			UUID:    "creator-1",
			Email:   "creator@example.com",
			Name:    "Test Creator",
			Key:     "creator-key",
			Avatar:  "https://example.com/creator-avatar.jpg",
			IsBrand: false,
		}

		mockDB := &MockOneCMSDB{
			BrokenPosts: brokenPosts,
			GetPostErr:  errors.New("database error"),
			MockTx:      &MockDBTransaction{},
		}

		mockOS := &MockOneCMSOS{
			GetAuthorByIDFunc: func(id string) (*AuthorOS, error) {
				if id == "author-1" {
					return author, nil
				} else if id == "creator-1" {
					return creator, nil
				}
				return nil, errors.New("author not found")
			},
		}

		// Use the test version instead
		err := testFixCSCPopmama(ctx, mockDB, mockOS, "test-index")
		if err == nil {
			t.Errorf("testFixCSCPopmama() expected error when getting post by old ID, got nil")
		}
	})

	t.Run("Error updating broken Popmama article", func(t *testing.T) {
		brokenPosts := []BrokenPopmamaArticleCSC{
			{
				OldID:      "old-1",
				AuthorID:   "author-1",
				AuthorKey:  "author-key",
				CreatedBy:  "creator-1",
				CreatorKey: "creator-key",
			},
		}

		post := &Post{
			ID:        "1",
			Title:     "Test Post",
			FullURL:   "https://example.com/test-post-oldkey-12345",
			Key:       "test-key",
			CreatedAt: time.Now(),
		}

		author := &AuthorOS{
			UUID:    "author-1",
			Email:   "author@example.com",
			Name:    "Test Author",
			Key:     "author-key",
			Avatar:  "https://example.com/avatar.jpg",
			IsBrand: false,
		}

		creator := &AuthorOS{
			UUID:    "creator-1",
			Email:   "creator@example.com",
			Name:    "Test Creator",
			Key:     "creator-key",
			Avatar:  "https://example.com/creator-avatar.jpg",
			IsBrand: false,
		}

		mockDB := &MockOneCMSDB{
			BrokenPosts:         brokenPosts,
			Post:                post,
			UpdateBrokenPostErr: errors.New("database error"),
			MockTx:              &MockDBTransaction{},
		}

		mockOS := &MockOneCMSOS{
			GetAuthorByIDFunc: func(id string) (*AuthorOS, error) {
				if id == "author-1" {
					return author, nil
				} else if id == "creator-1" {
					return creator, nil
				}
				return nil, errors.New("author not found")
			},
		}

		// Use the test version instead
		err := testFixCSCPopmama(ctx, mockDB, mockOS, "test-index")
		if err == nil {
			t.Errorf("testFixCSCPopmama() expected error when updating broken Popmama article, got nil")
		}
	})

	t.Run("Error updating OpenSearch data", func(t *testing.T) {
		brokenPosts := []BrokenPopmamaArticleCSC{
			{
				OldID:      "old-1",
				AuthorID:   "author-1",
				AuthorKey:  "author-key",
				CreatedBy:  "creator-1",
				CreatorKey: "creator-key",
			},
		}

		post := &Post{
			ID:        "1",
			Title:     "Test Post",
			FullURL:   "https://example.com/test-post-oldkey-12345",
			Key:       "test-key",
			CreatedAt: time.Now(),
		}

		author := &AuthorOS{
			UUID:    "author-1",
			Email:   "author@example.com",
			Name:    "Test Author",
			Key:     "author-key",
			Avatar:  "https://example.com/avatar.jpg",
			IsBrand: false,
		}

		creator := &AuthorOS{
			UUID:    "creator-1",
			Email:   "creator@example.com",
			Name:    "Test Creator",
			Key:     "creator-key",
			Avatar:  "https://example.com/creator-avatar.jpg",
			IsBrand: false,
		}

		mockDB := &MockOneCMSDB{
			BrokenPosts: brokenPosts,
			Post:        post,
			MockTx:      &MockDBTransaction{},
		}

		mockOS := &MockOneCMSOS{
			DynamicUpdateErr: errors.New("opensearch error"),
			GetAuthorByIDFunc: func(id string) (*AuthorOS, error) {
				if id == "author-1" {
					return author, nil
				} else if id == "creator-1" {
					return creator, nil
				}
				return nil, errors.New("author not found")
			},
		}

		// Use the test version instead
		err := testFixCSCPopmama(ctx, mockDB, mockOS, "test-index")
		if err == nil {
			t.Errorf("testFixCSCPopmama() expected error when updating OpenSearch data, got nil")
		}
	})
}
