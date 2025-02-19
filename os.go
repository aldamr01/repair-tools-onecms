package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go"

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
