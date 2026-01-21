package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esapi"
)

var EsClient *elasticsearch.Client

const wordDescIndex = "word_desc"

func InitEs() error {
	cfg := elasticsearch.Config{
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second,
			DialContext:           (&net.Dialer{Timeout: time.Second}).DialContext,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}
	var err error
	EsClient, err = elasticsearch.NewClient(cfg)
	if err != nil {
		return err
	}
	res, err := EsClient.Info()
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// fmt.Println(res)
	return nil
}

// index or update word desc
// test over
func IndexWordDesc(word *wordDesc) error {
	if EsClient == nil {
		return errors.New("es client not initialized")
	}
	body, err := json.Marshal(word)
	if err != nil {
		return err
	}
	res, err := EsClient.Index(
		wordDescIndex,
		bytes.NewReader(body),
		EsClient.Index.WithDocumentID(strconv.FormatInt(word.WordID, 10)),
		EsClient.Index.WithRefresh("true"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return parseEsError(res)
	}
	return nil
}

// update synonyms for example, set only synonyms
// todo test over
func UpdateWordDesc(word *wordDesc) error {
	if EsClient == nil {
		return errors.New("es client not initialized")
	}
	doc := map[string]interface{}{
		"doc": word,
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	res, err := EsClient.Update(
		wordDescIndex,
		strconv.FormatInt(word.WordID, 10),
		bytes.NewReader(body),
		EsClient.Update.WithRefresh("true"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return parseEsError(res)
	}
	return nil
}

func DeleteWordDesc(wordID int64) error {
	if EsClient == nil {
		return errors.New("es client not initialized")
	}
	res, err := EsClient.Delete(
		wordDescIndex,
		strconv.FormatInt(wordID, 10),
		EsClient.Delete.WithRefresh("true"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return parseEsError(res)
	}
	return nil
}

func SearchWordDescByWord(word string, size int) ([]wordDesc, error) {
	if EsClient == nil {
		return nil, errors.New("es client not initialized")
	}
	if size <= 0 {
		size = 10
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"word": word,
			},
		},
		"size": size,
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	res, err := EsClient.Search(
		EsClient.Search.WithIndex(wordDescIndex),
		EsClient.Search.WithBody(bytes.NewReader(body)),
		EsClient.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, parseEsError(res)
	}
	var resp struct {
		Hits struct {
			Hits []struct {
				Source wordDesc `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, err
	}
	results := make([]wordDesc, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		results = append(results, hit.Source)
	}
	return results, nil
}

func SearchWordDescFuzzy(word string, size int) ([]wordDesc, error) {
	if EsClient == nil {
		return nil, errors.New("es client not initialized")
	}
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, errors.New("empty search word")
	}
	if size <= 0 {
		size = 10
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"word": map[string]interface{}{
					"query":          word,
					"fuzziness":      "AUTO",
					"prefix_length":  1,
					"max_expansions": 50,
				},
			},
		},
		"size": size,
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	res, err := EsClient.Search(
		EsClient.Search.WithIndex(wordDescIndex),
		EsClient.Search.WithBody(bytes.NewReader(body)),
		EsClient.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, parseEsError(res)
	}
	var resp struct {
		Hits struct {
			Hits []struct {
				Source wordDesc `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, err
	}
	results := make([]wordDesc, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		results = append(results, hit.Source)
	}
	return results, nil
}

//第一次 search 拿到 scroll_id + 第一批数据，之后用 scroll_id 一批批拉完所有文档，只提取 _id。
func SearchAllWordIDs(batchSize int) ([]int64, error) {
	if EsClient == nil {
		return nil, errors.New("es client not initialized")
	}
	if batchSize <= 0 {
		batchSize = 1000
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": batchSize,
		"sort": []interface{}{
			map[string]interface{}{
				"_doc": "asc",
			},
		},
		"_source": false,
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	res, err := EsClient.Search(
		EsClient.Search.WithIndex(wordDescIndex),
		EsClient.Search.WithBody(bytes.NewReader(body)),
		EsClient.Search.WithScroll(time.Minute),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, parseEsError(res)
	}
	var resp struct {
		ScrollID string `json:"_scroll_id"`
		Hits     struct {
			Hits []struct {
				ID string `json:"_id"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, err
	}
	scrollID := resp.ScrollID
	defer func() {
		if scrollID == "" {
			return
		}
		_, _ = EsClient.ClearScroll(
			EsClient.ClearScroll.WithScrollID(scrollID),
		)
	}()
	ids := make([]int64, 0, len(resp.Hits.Hits))
	appendIDs := func(hits []struct {
		ID string `json:"_id"`
	}) error {
		for _, hit := range hits {
			id, err := strconv.ParseInt(hit.ID, 10, 64)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	}
	if err := appendIDs(resp.Hits.Hits); err != nil {
		return nil, err
	}
	for len(resp.Hits.Hits) > 0 {
		res, err = EsClient.Scroll(
			EsClient.Scroll.WithScrollID(scrollID),
			EsClient.Scroll.WithScroll(time.Minute),
		)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.IsError() {
			return nil, parseEsError(res)
		}
		resp = struct {
			ScrollID string `json:"_scroll_id"`
			Hits     struct {
				Hits []struct {
					ID string `json:"_id"`
				} `json:"hits"`
			} `json:"hits"`
		}{}
		if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
			return nil, err
		}
		if resp.ScrollID != "" {
			scrollID = resp.ScrollID
		}
		if err := appendIDs(resp.Hits.Hits); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

func parseEsError(res *esapi.Response) error {
	body, _ := io.ReadAll(res.Body)
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = "empty response body"
	}
	return fmt.Errorf("es error: status=%s body=%s", res.Status(), msg)
}
