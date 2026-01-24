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

var esClient *EsClient

type EsClient struct {
	client *elasticsearch.Client
}

const wordDescIndex = "word_desc"

func InitEs() error {
	esClient = &EsClient{}
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
	esClient.client, err = elasticsearch.NewClient(cfg)
	if err != nil {
		return err
	}
	res, err := esClient.client.Info()
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// fmt.Println(res)
	return nil
}

// index or update word desc
// test over
func (es *EsClient) IndexWordDesc(word *wordDesc) error {
	if es.client == nil {
		return errors.New("es client not initialized")
	}
	body, err := json.Marshal(word)
	if err != nil {
		return err
	}
	res, err := es.client.Index(
		wordDescIndex,
		bytes.NewReader(body),
		es.client.Index.WithDocumentID(strconv.FormatInt(word.WordID, 10)),
		es.client.Index.WithRefresh("true"),
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
// test over
func (es *EsClient) UpdateWordDesc(word *wordDesc) error {
	if es.client == nil {
		return errors.New("es client not initialized")
	}
	doc := map[string]interface{}{
		"doc": word,
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	res, err := es.client.Update(
		wordDescIndex,
		strconv.FormatInt(word.WordID, 10),
		bytes.NewReader(body),
		es.client.Update.WithRefresh("true"),
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

// test over
func (es *EsClient) DeleteWordById(wordID int64) error {
	if es.client == nil {
		return errors.New("es client not initialized")
	}
	res, err := es.client.Delete(
		wordDescIndex,
		strconv.FormatInt(wordID, 10),
		es.client.Delete.WithRefresh("true"),
	)
	if res.StatusCode == 404 {
		return nil
	}
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return parseEsError(res)
	}
	return nil
}

// 按 word 精确删除（使用 word.keyword）
func (es *EsClient) DeleteWordByName(word string) error {
	if es.client == nil {
		return errors.New("es client not initialized")
	}
	word = strings.TrimSpace(word)
	if word == "" {
		return errors.New("empty word")
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"word.keyword": word,
			},
		},
	}

	body, err := json.Marshal(query)
	if err != nil {
		return err
	}

	res, err := es.client.DeleteByQuery(
		[]string{wordDescIndex},
		bytes.NewReader(body),
		es.client.DeleteByQuery.WithRefresh(true),
	)
	//如果删除的目标不存在，则直接忽略
	if res.StatusCode == 404 {
		return nil
	}
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return parseEsError(res)
	}
	return nil
}

// 以这个接口为base,设计其他搜索接口
// 1. 拼写错误
// 2. 前缀搜索
// 3. 汉语意思搜索
func (es *EsClient) searchBaseOnQuery(query map[string]interface{}) ([]wordDesc, error) {
	if es.client == nil {
		return nil, errors.New("es client not initialized")
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	res, err := es.client.Search(
		es.client.Search.WithIndex(wordDescIndex),
		es.client.Search.WithBody(bytes.NewReader(body)),
		es.client.Search.WithTrackTotalHits(true),
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

// test over
func (es *EsClient) SearchWordDescFuzzy(word string, size int) ([]wordDesc, error) {
	if es.client == nil {
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
	return es.searchBaseOnQuery(query)
}

// test over
func (es *EsClient) SearchWordDescByWord(word string) ([]wordDesc, error) {
	if es.client == nil {
		return nil, errors.New("es client not initialized")
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"word": word,
			},
		},
		"size": 10,
	}
	return es.searchBaseOnQuery(query)
}

// test over
func (es *EsClient) SearchWordDescByWordPrefix(word string) ([]wordDesc, error) {
	if es.client == nil {
		return nil, errors.New("es client not initialized")
	}
	word = strings.TrimSpace(word)
	if word == "" {
		return nil, errors.New("empty search word")
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"prefix": map[string]interface{}{
				"word": map[string]interface{}{
					"value": word,
				},
			},
		},
		"size": 10,
	}
	return es.searchBaseOnQuery(query)
}

// test over
func (es *EsClient) SearchWordDescByChineseMeaning(meaning string) ([]wordDesc, error) {
	if es.client == nil {
		return nil, errors.New("es client not initialized")
	}
	meaning = strings.TrimSpace(meaning)
	if meaning == "" {
		return nil, errors.New("empty search meaning")
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"definitions.meaning": meaning,
			},
		},
		"size": 10,
	}
	return es.searchBaseOnQuery(query)
}

// test over
// 第一次 search 拿到 scroll_id + 第一批数据，之后用 scroll_id 一批批拉完所有文档，提取 _id 和 word。
func (es *EsClient) SearchAllWordIDs(batchSize int) (map[int64]string, error) {
	if es.client == nil {
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
		"_source": []string{"word"},
	}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	res, err := es.client.Search(
		es.client.Search.WithIndex(wordDescIndex),
		es.client.Search.WithBody(bytes.NewReader(body)),
		es.client.Search.WithScroll(time.Minute),
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
				ID     string `json:"_id"`
				Source struct {
					Word string `json:"word"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, err
	}
	scrollID := resp.ScrollID
	// ES 的 scroll 会在服务器端保持一个游标（scroll ID），不清理会占用资源。
	// 这段 defer 在函数结束时调用 ClearScroll，告诉 ES 释放这个 scroll：
	// 防止内存/资源泄露
	// 也是 ES 官方建议的做法
	defer func() {
		if scrollID == "" {
			return
		}
		_, _ = es.client.ClearScroll(
			es.client.ClearScroll.WithScrollID(scrollID),
		)
	}()
	idToWord := make(map[int64]string, len(resp.Hits.Hits))
	appendHits := func(hits []struct {
		ID     string `json:"_id"`
		Source struct {
			Word string `json:"word"`
		} `json:"_source"`
	}) error {
		for _, hit := range hits {
			id, err := strconv.ParseInt(hit.ID, 10, 64)
			if err != nil {
				return err
			}
			idToWord[id] = hit.Source.Word
		}
		return nil
	}
	if err := appendHits(resp.Hits.Hits); err != nil {
		return nil, err
	}
	for len(resp.Hits.Hits) > 0 {
		res, err = es.client.Scroll(
			es.client.Scroll.WithScrollID(scrollID),
			es.client.Scroll.WithScroll(time.Minute),
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
					ID     string `json:"_id"`
					Source struct {
						Word string `json:"word"`
					} `json:"_source"`
				} `json:"hits"`
			} `json:"hits"`
		}{}
		if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
			return nil, err
		}
		if resp.ScrollID != "" {
			scrollID = resp.ScrollID
		}
		if err := appendHits(resp.Hits.Hits); err != nil {
			return nil, err
		}
	}
	return idToWord, nil
}

func parseEsError(res *esapi.Response) error {
	body, _ := io.ReadAll(res.Body)
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = "empty response body"
	}
	return fmt.Errorf("es error: status=%s body=%s", res.Status(), msg)
}
