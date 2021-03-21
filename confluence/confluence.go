// Package confluence provides functionality for interacting with the confluence APIClient
// Specifically managing pages
package confluence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/xiatechs/markdown-to-confluence/markdown"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

// CreatePage in confluence
// todo: function not tested live on confluence yet! test written on expected results
func (a *APIClient) CreatePage(contents *markdown.FileContents) error {
	newPageContents, err := json.Marshal(contents.Body)
	if err != nil {
		return err
	}

	URL := fmt.Sprintf("%s/wiki/rest/api/content", a.BaseURL) // todo: might need space specification in url

	req, err := retryablehttp.NewRequest(http.MethodPost, URL, newPageContents)
	if err != nil {
		return err
	}

	req.SetBasicAuth(a.Username, a.Password)

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := a.Client.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create confluence page: %s", resp.Status)
	}

	return nil
}

// UpdatePage updates a confluence page with our newly created data and increases the
// version by 1 each time.
func (a *APIClient) UpdatePage(pageID int, pageVersion int64, pageContents *markdown.FileContents) error {
	fmt.Println("running update now....") //todo: remove

	newPageJSON := PutPageContent{
		Type:  "page",
		Title: pageContents.MetaData["title"].(string),
		Version: VersionObj{
			Number: int(pageVersion) + 1,
		},
		Body: BodyObj{
			Storage: StorageObj{
				Value:          string(pageContents.Body),
				Representation: "storage",
			},
		},
	}

	URL := fmt.Sprintf("%s/wiki/rest/api/content/%d", a.BaseURL, pageID)

	b, err := json.Marshal(newPageJSON)
	if err != nil {
		return err
	}

	fmt.Println(string(b)) //todo:remove

	req, err := retryablehttp.NewRequest(http.MethodPut, URL, bytes.NewBuffer(b))
	if err != nil {
		log.Println(err)
		return err
	}

	req.SetBasicAuth(a.Username, a.Password)

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Client.Do(req)
	if err != nil {
		fmt.Println("error was: ", resp.Status, err)
		return fmt.Errorf("failed to do the request: %w", err)
	}

	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error ioutil", err)
	}

	fmt.Println("response: ", string(r))

	defer func() { _ = resp.Body.Close() }()

	return nil
}

// FindPage in confluence
// Docs for this API endpoint are here
// https://developer.atlassian.com/cloud/confluence/rest/api-group-content/#api-api-content-get
func (a *APIClient) FindPage(title string) (int, int64, bool, error) {
	lookUpURL := fmt.Sprintf("%s/wiki/rest/api/content?expand=version&type=page&spaceKey=%s&title=%s",
		a.BaseURL, a.Space, title)

	req, err := retryablehttp.NewRequest(http.MethodGet, lookUpURL, nil)
	if err != nil {
		return 0, 0, false, err
	}

	req.SetBasicAuth(a.Username, a.Password)

	resp, err := a.Client.Do(req)
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to do the request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	r := findPageResult{}

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, 0, false, err
	}

	spew.Dump(r) // todo remove

	if len(r.Results) == 0 {
		return 0, 0, false, fmt.Errorf("no page present")
	}

	pageID, err := strconv.Atoi(r.Results[0].ID)
	if err != nil {
		fmt.Println("error converting ID to int value")

		return 0, 0, false, err
	}

	return pageID, r.Results[0].Version.Number, true, nil
}
