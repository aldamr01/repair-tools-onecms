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
	BeginTx(ctx context.Context) (*sql.Tx, error)
	Commit(ctx context.Context, tx *sql.Tx) error
	Rollback(ctx context.Context, tx *sql.Tx) error
	GetPostsByCreatedAt(ctx context.Context, startAt, endat string) ([]Post, error)
	GetBrokenPopmamaArticleCSC(ctx context.Context) ([]BrokenPopmamaArticleCSC, error)
	GetAuthorKeyByPostID(ctx context.Context, postID string) (string, error)
	UpdateArticleURLByID(ctx context.Context, postID, fixedURL string) error
	GetPostByOldIDAndPublisher(ctx context.Context, oldID, publisher string) (*Post, error)
	UpdateBrokenPopmamaArticleCSC(ctx context.Context, transactionDB *sql.Tx, postID string, post Post) error
	SetPostAuthor(ctx context.Context, transactionDB *sql.Tx, postID string, authorID string, orderNumber int) error
	FlushPostAuthors(ctx context.Context, transactionDB *sql.Tx, postID string) error
}

type oneCMSDB struct {
	dbClient sqlx.DB
}

func NewOneCMSDB(client sqlx.DB) OneCMSDB {
	return &oneCMSDB{
		dbClient: client,
	}
}

func (oneDB *oneCMSDB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, err := oneDB.dbClient.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (oneDB *oneCMSDB) Commit(ctx context.Context, tx *sql.Tx) error {
	err := tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (oneDB *oneCMSDB) Rollback(ctx context.Context, tx *sql.Tx) error {
	err := tx.Rollback()
	if err != nil {
		return err
	}

	return nil
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

	_, err := oneDB.dbClient.ExecContext(ctx, query, fixedURL, postID)

	return err
}

func (oneDB *oneCMSDB) GetPostByOldIDAndPublisher(ctx context.Context, oldID, publisher string) (*Post, error) {
	var post *Post

	query := `
		SELECT * FROM posts WHERE old_id = $1 AND publisher = $2
	`

	c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := oneDB.dbClient.GetContext(c, &post, query, oldID, publisher)
	if err != nil {
		return nil, err
	}

	return post, nil
}

func (oneDB *oneCMSDB) GetBrokenPopmamaArticleCSC(ctx context.Context) ([]BrokenPopmamaArticleCSC, error) {
	query := `
		SELECT 
			p.old_id,
			p.author_id,
			p.author_key,
			p.created_by,
			p.creator_key
		FROM temp_popmama_csc
	`

	rows, err := oneDB.dbClient.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	items := []BrokenPopmamaArticleCSC{}
	for rows.Next() {
		var post BrokenPopmamaArticleCSC
		err := rows.Scan(
			&post.OldID,
			&post.AuthorID,
			&post.AuthorKey,
			&post.CreatedBy,
			&post.CreatorKey,
		)

		if err != nil {
			return nil, err
		}

		items = append(items, post)
	}

	return items, nil
}

func (oneDB *oneCMSDB) UpdateBrokenPopmamaArticleCSC(ctx context.Context, transactionDB *sql.Tx, postID string, post Post) error {
	query := `
		UPDATE
			posts
		SET 
			full_url = $1,
			author_id = $2
		WHERE id = $3
	`

	_, err := transactionDB.ExecContext(ctx, query, post.FullURL, post.AuthorID, postID)
	if err != nil {
		transactionDB.Rollback()
	}

	return err
}

func (oneDB *oneCMSDB) SetPostAuthor(ctx context.Context, transactionDB *sql.Tx, postID string, authorID string, orderNumber int) error {
	postAuthorValues := []interface{}{postID, authorID, orderNumber}
	postAuthorQuery := `INSERT INTO post_authors
	(
		post_id,
		author_id,
		order_number
	)
	VALUES ($1, $2, $3)`

	if _, err := transactionDB.ExecContext(ctx, postAuthorQuery, postAuthorValues...); err != nil {
		transactionDB.Rollback()
		return err
	}

	return nil
}

func (oneDB *oneCMSDB) FlushPostAuthors(ctx context.Context, transactionDB *sql.Tx, postID string) error {

	queryDel := `DELETE FROM post_authors
    	WHERE post_id = $1`

	_, err := transactionDB.ExecContext(ctx, queryDel, postID)
	if err != nil {
		transactionDB.Rollback()
	}

	return err
}
