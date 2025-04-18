package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
)

func fixURL(ctx context.Context, onecmsDB OneCMSDB, onecmsOS OneCMSOS, startAt, endAt, osIndex string) error {
	fmt.Printf("🔁 Calculating posts based from created at %v to %v\n", startAt, endAt)
	posts, err := onecmsDB.GetPostsByCreatedAt(ctx, startAt, endAt)
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
		ArticleURL    string `json:"article_url"`
		ArticleURLAMP string `json:"article_url_amp"`
	}

	for i, chunk := range chunks {
		fmt.Printf("🔁 [%d/%d] Running chunk...\n", i+1, chunkLength)
		cl := len(chunk)

		for i, post := range chunk {
			fmt.Printf("\n\t[%d/%d] Fixing post url...", i+1, cl)

			authorKey, err := onecmsDB.GetAuthorKeyByPostID(ctx, post.ID)
			if err != nil || authorKey == "" {
				msg := fmt.Errorf("\n\t❌ Cannot find author for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post url wiith ID %s, caused by: %s. Error: %s", post.ID, msg, err.Error()))
				continue
			}

			currentURL := post.FullURL
			fixedURL, err := FixURL(currentURL, authorKey)
			if err != nil {
				msg := fmt.Errorf("\n\t❌ Failed fixing url for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post url wiith ID %s, caused by: %s. Error: %s", post.ID, msg, err.Error()))
				continue
			}

			if err := onecmsDB.UpdateArticleURLByID(ctx, post.ID, fixedURL); err != nil {
				msg := fmt.Errorf("\n\t❌ Failed updating DB data for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post url wiith ID %s, caused by: %s. Error: %s", post.ID, msg, err.Error()))
				continue
			}

			osData := postOSStructure{
				ArticleURL:    fixedURL,
				ArticleURLAMP: fixedURL + "/amp",
			}

			if err := onecmsOS.DynamicUpdate(osData, post.ID, osIndex); err != nil {
				msg := fmt.Errorf("\n\t❌ Failed updating OS data for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post url wiith ID %s, caused by: %s. Error: %s", post.ID, msg, err.Error()))
				continue
			}

			fmt.Printf("\n\t 🧑🏾‍💻 Author key: %s", authorKey)
			fmt.Printf("\n\t 🌏 URL: %s -> %s", currentURL, fixedURL)
			fmt.Printf("\n\t ✅ Success fixing post url with id %s ✔️\n", post.ID)
		}

		fmt.Println("-----🚀-----")
	}

	fmt.Printf("\n🚚 UNFIXED: %v", PrettyF(unfixedPosts))

	if len(unfixedPosts) > 0 {
		return fmt.Errorf("\n❗Few total error: %d \n 🚚 UNFIXED: %v", len(unfixedPosts), PrettyF(unfixedPosts))
	}

	return nil
}

func fixCSCPopmama(ctx context.Context, onecmsDB OneCMSDB, onecmsOS OneCMSOS, osIndex string) error {
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

			// TODO: Fixing section here
			transactionDB, err := onecmsDB.BeginTx(ctx)
			if err != nil {
				msg := fmt.Errorf("\n\t❌ Failed starting transaction.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			fixedURL, err := FixURL(postExisting.FullURL, postAuthor.Key)
			if err != nil {
				transactionDB.Rollback()
				msg := fmt.Errorf("\n\t❌ Failed generate fixed url for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			postExisting.FullURL = fixedURL
			postExisting.CreatedBy = postCreator.Key
			postExisting.AuthorID = postAuthor.Key

			if err := onecmsDB.UpdateBrokenPopmamaArticleCSC(ctx, transactionDB, post.OldID, *postExisting); err != nil {
				msg := fmt.Errorf("\n\t❌ Failed updating DB data for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			if err := onecmsDB.FlushPostAuthors(ctx, transactionDB, postExisting.ID); err != nil {
				msg := fmt.Errorf("\n\t❌ Failed flushing post authors for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			if err := onecmsDB.SetPostAuthor(ctx, transactionDB, postExisting.ID, postAuthor.Key, 0); err != nil {
				msg := fmt.Errorf("\n\t❌ Failed setting post author for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			osData := postOSStructure{
				ArticleURL:    fixedURL,
				ArticleURLAMP: fixedURL + "/amp",
				Authors:       []AuthorOS{*postAuthor},
			}

			if err := onecmsOS.DynamicUpdate(osData, postExisting.ID, osIndex); err != nil {
				transactionDB.Rollback()
				msg := fmt.Errorf("\n\t❌ Failed updating OS data for this post.")
				unfixedPosts = append(unfixedPosts, fmt.Sprintf("Error fixing post with old id: %s, caused by: %s. Error: %s", post.OldID, msg, err.Error()))
				continue
			}

			if err := transactionDB.Commit(); err != nil {
				msg := fmt.Errorf("\n\t❌ Failed committing transaction.")
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
