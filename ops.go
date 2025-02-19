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

	ppppps := []map[string]interface{}{}

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

	fmt.Println(">>>>")
	PrettyPrint(ppppps)
	fmt.Println("<<<<")
	fmt.Printf("\n🚚 UNFIXED: %v", PrettyF(unfixedPosts))

	if len(unfixedPosts) > 0 {
		return fmt.Errorf("\n❗Few total error: %d", len(unfixedPosts))
	}

	return nil
}
