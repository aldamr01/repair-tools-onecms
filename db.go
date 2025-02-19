package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func GetDBConnection(DSN string) (*sqlx.DB, error) {
	db, err := sql.Open("postgres", DSN)

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("Database ping failed with error: %s", err.Error())
	}

	fmt.Println("✅ Database is connected successfully!")

	return sqlx.NewDb(db, "postgres"), err
}

type OneCMSDB interface {
	GetPostsByCreatedAt(ctx context.Context, startAt, endat string) ([]Post, error)
	GetAuthorKeyByPostID(ctx context.Context, postID string) (string, error)
	UpdateArticleURLByID(ctx context.Context, postID, fixedURL string) error
}

type oneCMSDB struct {
	dbClient sqlx.DB
}

func NewOneCMSDB(client sqlx.DB) OneCMSDB {
	return &oneCMSDB{
		dbClient: client,
	}
}

func (oneDB *oneCMSDB) GetPostsByCreatedAt(ctx context.Context, startAt, endAt string) ([]Post, error) {
	query := `			
		SELECT 
			p.id,
			p.title,
			p.full_url,			
			p.key,
			p.created_at
		FROM posts p		
		WHERE p.created_at >= $1 
			AND p.created_at <= $2
		ORDER BY p.created_at
	`
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := oneDB.dbClient.QueryContext(c, query, startAt, endAt)
	if err != nil {
		return nil, err
	}

	items := []Post{}
	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.ID,
			&post.Title,
			&post.FullURL,
			&post.Key,
			&post.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		items = append(items, post)
	}

	return items, nil
}

func (oneDB *oneCMSDB) GetAuthorKeyByPostID(ctx context.Context, postID string) (string, error) {
	var authorKey string

	query := `			
		SELECT u."key"
		FROM post_authors pa 
		LEFT JOIN users u ON u.id = pa.author_id  
		WHERE pa.post_id = $1
		ORDER BY pa.order_number ASC
		LIMIT 1
	`
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := oneDB.dbClient.GetContext(c, &authorKey, query, postID)
	if err != nil {
		return "", err
	}

	return authorKey, nil
}

func (oneDB *oneCMSDB) UpdateArticleURLByID(ctx context.Context, postID, fixedURL string) error {
	query := `
		UPDATE
			posts
		SET full_url = $1
		WHERE id = $2
	`
	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := oneDB.dbClient.ExecContext(c, query, fixedURL, postID)

	return err
}
