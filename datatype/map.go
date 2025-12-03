package datatype

import "encoding/json"

type Context map[string]interface{}
type ContextObject map[string]map[string]interface{}

type JsonObject map[string]interface{}
type DataMap map[string]interface{}

func (c *Context) ToString() string {
	jsonData, _ := json.MarshalIndent(c.ToMap(), "", " ")

	return string(jsonData)
}

func (c *Context) ToMap() map[string]interface{} {
	return *c
}
