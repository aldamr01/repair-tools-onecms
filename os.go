package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"

	// _ empty import used to initilize driver
	_ "github.com/lib/pq"
)

// OnecmsSQLConn is a function to open database connection pool
func GetOSConnection(host, username, password string) (*opensearch.Client, error) {

	client, err := opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Addresses: []string{host},
		Username:  username,
		Password:  password,
	})

	res, err := client.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OpenSearch: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenSearch ping failed with status: %d", res.StatusCode)
	}

	fmt.Println("✅ OpenSearch is connected successfully!")

	return client, err
}

type OneCMSOS interface {
	DynamicUpdate(data interface{}, docID, index string) error
	GetAuthorByID(authorID string) (*AuthorOS, error)
}

type oneCMSOS struct {
	osClient *opensearch.Client
}

func NewOneCMSOS(client *opensearch.Client) *oneCMSOS {
	return &oneCMSOS{
		osClient: client,
	}
}

func (oneOS *oneCMSOS) DynamicUpdate(data interface{}, docID, index string) error {

	docData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	json := `{ "update" : {"_id" : "%s", "_index" : "%s" } }
    { "doc" : %s }`

	query := fmt.Sprintf(json, docID, index, string(docData))

	bulkResponse, err := oneOS.osClient.Bulk(strings.NewReader(string(query) + "\n"))
	if err != nil {
		return err
	}

	if bulkResponse.IsError() {
		return errors.New(bulkResponse.String())
	}

	return nil
}

func (oneOS *oneCMSOS) GetAuthorByID(authorID string) (*AuthorOS, error) {

	authorIndex := "one-author-index"
	var author *AuthorOS

	osGet := opensearchapi.GetRequest{
		Index:      authorIndex,
		DocumentID: authorID,
	}

	getResponse, err := osGet.Do(context.Background(), oneOS.osClient)
	if err != nil {
		return author, err
	}
	defer getResponse.Body.Close()

	result := AuthorGetResult{}
	if err := json.NewDecoder(getResponse.Body).Decode(&result); err != nil {
		return author, err
	}

	if !result.Found {
		return nil, errors.New("post not found")
	}

	author = result.Source

	return author, nil
}
