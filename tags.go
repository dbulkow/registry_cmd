package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Tags struct {
	tags []string
}

func (t *Tags) Method() string { return http.MethodGet }

func (t *Tags) SetHeaders(hdr *http.Header) {}

func (t *Tags) UnmarshalJSON(b []byte) error {
	type Tags struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}

	tags := &Tags{}

	err := json.Unmarshal(b, &tags)
	if err != nil {
		return fmt.Errorf("unmarshal: %v", err)
	}

	t.tags = tags.Tags

	return nil
}

func (t *Tags) ExtractHeaders(hdr *http.Header) {}

func tags(conn *http.Client, url, image string) ([]string, error) {
	t := &Tags{}

	err := get(conn, url+"/v2/"+image+"/tags/list", t)
	if err != nil {
		return nil, err
	}

	return t.tags, nil
}
