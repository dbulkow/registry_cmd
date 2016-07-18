package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Catalog struct {
	images []string
}

func (c *Catalog) Method() string { return http.MethodGet }

func (c *Catalog) SetHeaders(hdr *http.Header) {}

func (c *Catalog) UnmarshalJSON(b []byte) error {
	type Catalog struct {
		Repositories []string `json:"repositories"`
	}

	cat := &Catalog{}

	err := json.Unmarshal(b, &cat)
	if err != nil {
		return fmt.Errorf("unmarshal: %v", err)
	}

	c.images = cat.Repositories

	return nil
}

func (c *Catalog) ExtractHeaders(hdr *http.Header) {}

func catalog(conn *http.Client, url string) ([]string, error) {
	c := &Catalog{}

	err := get(conn, url+"/v2/_catalog", c)
	if err != nil {
		return nil, err
	}

	return c.images, nil
}
