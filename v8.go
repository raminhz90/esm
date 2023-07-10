package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	log "github.com/cihub/seelog"
)

// ESAPIV8 is a struct representing an Elasticsearch API version 8.
type ESAPIV8 struct {
	ESAPIV7
}

// NewScroll creates a  scroll in Elasticsearch API version 8.
func (s *ESAPIV8) NewScroll(indexNames string, scrollTime string, docBufferCount int, query string, slicedId, maxSlicedCount int, fields string) (scroll interface{}, err error) {
	// Build the URL for the Elasticsearch search API with scroll.
	url := fmt.Sprintf("%s/%s/_search?scroll=%s&size=%d", s.Host, indexNames, scrollTime, docBufferCount)

	// Create the body of the request if necessary.
	jsonBody := createJSONBody(query, maxSlicedCount, fields, slicedId)

	// Send the POST request.
	resp, body, errs := Post(url, s.Auth, jsonBody, s.HttpProxy)
	defer closeResponse(resp)

	if errs != nil {
		log.Error(errs)
		return nil, errs[0]
	}

	if resp.StatusCode != 200 {
		return nil, errors.New(body)
	}

	log.Trace("new scroll,", body)

	// Parse the response.
	scroll = &ScrollV7{}
	err = DecodeJson(body, scroll)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return scroll, err
}
func createJSONBody(query string, maxSlicedCount int, fields string, slicedId int) string {
	if len(query) == 0 && maxSlicedCount == 0 && len(fields) == 0 {
		return ""
	}

	queryBody := make(map[string]interface{})

	if len(fields) > 0 {
		if !strings.Contains(fields, ",") {
			queryBody["_source"] = fields
		} else {
			queryBody["_source"] = strings.Split(fields, ",")
		}
	}

	if len(query) > 0 {
		queryBody["query"] = map[string]interface{}{
			"query_string": map[string]interface{}{
				"query": query,
			},
		}
	}

	if maxSlicedCount > 1 {
		log.Tracef("sliced scroll, %d of %d", slicedId, maxSlicedCount)
		queryBody["slice"] = map[string]interface{}{
			"id":  slicedId,
			"max": maxSlicedCount,
		}
	}

	jsonArray, err := json.Marshal(queryBody)
	if err != nil {
		log.Error(err)
		return ""
	}

	return string(jsonArray)
}

// closeResponse closes the HTTP response body.
func closeResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
func (s *ESAPIV8) NextScroll(scrollTime string, scrollId string) (interface{}, error) {
	id := bytes.NewBufferString(scrollId)

	url := fmt.Sprintf("%s/_search/scroll?scroll=%s&scroll_id=%s", s.Host, scrollTime, id)
	body, err := DoRequest(s.Compress, "GET", url, s.Auth, nil, s.HttpProxy)

	if err != nil {
		//log.Error(errs)
		return nil, err
	}
	// decode elasticsearch scroll response
	scroll := &ScrollV7{}
	err = DecodeJson(body, &scroll)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return scroll, nil
}

func (s *ESAPIV8) GetIndexSettings(indexNames string) (*Indexes, error) {
	return s.ESAPIV0.GetIndexSettings(indexNames)
}

func (s *ESAPIV8) UpdateIndexSettings(indexName string, settings map[string]interface{}) error {
	return s.ESAPIV0.UpdateIndexSettings(indexName, settings)
}

func (s *ESAPIV8) GetIndexMappings(copyAllIndexes bool, indexNames string) (string, int, *Indexes, error) {
	url := fmt.Sprintf("%s/%s/_mapping", s.Host, indexNames)
	resp, body, errs := Get(url, s.Auth, s.HttpProxy)

	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		defer resp.Body.Close()
	}

	if errs != nil {
		log.Error(errs)
		return "", 0, nil, errs[0]
	}

	if resp.StatusCode != 200 {
		return "", 0, nil, errors.New(body)
	}

	idxs := Indexes{}
	er := DecodeJson(body, &idxs)

	if er != nil {
		log.Error(body)
		return "", 0, nil, er
	}

	// if _all indexes limit the list of indexes to only these that we kept
	// after looking at mappings
	if indexNames == "_all" {

		var newIndexes []string
		for name := range idxs {
			newIndexes = append(newIndexes, name)
		}
		indexNames = strings.Join(newIndexes, ",")

	} else if strings.Contains(indexNames, "*") || strings.Contains(indexNames, "?") {

		r, _ := regexp.Compile(indexNames)

		//check index patterns
		var newIndexes []string
		for name := range idxs {
			matched := r.MatchString(name)
			if matched {
				newIndexes = append(newIndexes, name)
			}
		}
		indexNames = strings.Join(newIndexes, ",")

	}

	i := 0
	// wrap in mappings if moving from super old es
	for name, idx := range idxs {
		i++
		fmt.Println(name)
		if _, ok := idx.(map[string]interface{})["mappings"]; !ok {
			(idxs)[name] = map[string]interface{}{
				"mappings": idx,
			}
		}
	}

	return indexNames, i, &idxs, nil
}

func (s *ESAPIV8) UpdateIndexMapping(indexName string, settings map[string]interface{}) error {

	log.Debug("start update mapping: ", indexName, settings)

	delete(settings, "dynamic_templates")

	log.Debug("start update mapping: ", indexName, ", ", settings)

	url := fmt.Sprintf("%s/%s/_mapping", s.Host, indexName)

	body := bytes.Buffer{}
	enc := json.NewEncoder(&body)
	enc.Encode(settings)
	res, err := Request("POST", url, s.Auth, &body, s.HttpProxy)
	if err != nil {
		log.Error(url)
		log.Error(body.String())
		log.Error(err, res)
		panic(err)
	}
	return nil
}
