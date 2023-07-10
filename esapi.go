package main

import "bytes"

type ESAPI interface{
	ClusterHealth() *ClusterHealth
	Bulk(data *bytes.Buffer)
	GetIndexSettings(indexNames string) (*Indexes, error)
	DeleteIndex(name string) (error)
	CreateIndex(name string,settings map[string]interface{}) (error)
	GetIndexMappings(copyAllIndexes bool,indexNames string)(string,int,*Indexes,error)
	UpdateIndexSettings(indexName string,settings map[string]interface{})(error)
	UpdateIndexMapping(indexName string,mappings map[string]interface{})(error)
	NewScroll(indexNames string,scrollTime string,docBufferCount int,query string, slicedId,maxSlicedCount int, fields string)(interface{}, error)
	NextScroll(scrollTime string,scrollId string)(interface{},error)
	Refresh(name string) (err error)
}
