package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/raminhz90/esm/util"
)

type ESAPIV6 struct {
	ESAPIV5
}

func (s *ESAPIV6) NewScroll(indexNames string, scrollTime string, docBufferCount int, query string, slicedId, maxSlicedCount int, fields string) (scroll interface{}, err error) {
	url := fmt.Sprintf("%s/%s/_search?scroll=%s&size=%d", s.Host, indexNames, scrollTime, docBufferCount)

	var jsonBody []byte
	if len(query) > 0 || maxSlicedCount > 0 || len(fields) > 0 {
		queryBody := map[string]interface{}{}

		if len(fields) > 0 {
			if !strings.Contains(fields, ",") {
				queryBody["_source"] = fields
			} else {
				queryBody["_source"] = strings.Split(fields, ",")
			}
		}

		if len(query) > 0 {
			queryBody["query"] = map[string]interface{}{}
			queryBody["query"].(map[string]interface{})["query_string"] = map[string]interface{}{}
			queryBody["query"].(map[string]interface{})["query_string"].(map[string]interface{})["query"] = query
		}

		if maxSlicedCount > 1 {
			log.Tracef("sliced scroll, %d of %d", slicedId, maxSlicedCount)
			queryBody["slice"] = map[string]interface{}{}
			queryBody["slice"].(map[string]interface{})["id"] = slicedId
			queryBody["slice"].(map[string]interface{})["max"] = maxSlicedCount
		}

		jsonBody, err = json.Marshal(queryBody)
		if err != nil {
			log.Error(err)

		}
	}

	body, err := DoRequest(s.Compress, "POST", url, s.Auth, jsonBody, s.HttpProxy)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	scroll = &Scroll{}
	err = DecodeJson(body, scroll)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return scroll, err
}

func (s *ESAPIV6) NextScroll(scrollTime string, scrollId string) (interface{}, error) {
	id := bytes.NewBufferString(scrollId)

	url := fmt.Sprintf("%s/_search/scroll?scroll=%s&scroll_id=%s", s.Host, scrollTime, id)
	body, _ := DoRequest(s.Compress, "GET", url, s.Auth, nil, s.HttpProxy)

	// decode elasticsearch scroll response
	scroll := &Scroll{}
	err := DecodeJson(body, &scroll)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return scroll, nil
}

func (s *ESAPIV6) GetIndexSettings(indexNames string) (*Indexes, error) {
	return s.ESAPIV0.GetIndexSettings(indexNames)
}

func (s *ESAPIV6) UpdateIndexSettings(indexName string, settings map[string]interface{}) error {
	return s.ESAPIV0.UpdateIndexSettings(indexName, settings)
}

func (s *ESAPIV6) GetIndexMappings(copyAllIndexes bool, indexNames string) (string, int, *Indexes, error) {
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
		if _, ok := idx.(map[string]interface{})["mappings"]; !ok {
			(idxs)[name] = map[string]interface{}{
				"mappings": idx,
			}
		}
	}

	return indexNames, i, &idxs, nil
}

func (s *ESAPIV6) UpdateIndexMapping(indexName string, settings map[string]interface{}) error {

	log.Debug("start update mapping: ", indexName, settings)

	delete(settings, "dynamic_templates")

	for name := range settings {

		log.Debug("start updating mappings: ", indexName, ", ", settings)

		url := fmt.Sprintf("%s/%s/%s/_mapping", s.Host, indexName, name)

		body := bytes.Buffer{}
		enc := json.NewEncoder(&body)
		enc.Encode(settings)
		res, err := Request("POST", url, s.Auth, &body, s.HttpProxy)
		if err != nil {
			log.Error(url)
			log.Error(util.ToJson(settings, false))
			log.Error(err, res)
			panic(err)
		}
	}
	return nil
}
