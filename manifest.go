package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Manifest struct {
	digest string
	blobs  []string
}

func (m *Manifest) Method() string { return http.MethodGet }

func (m *Manifest) SetHeaders(hdr *http.Header) {
	hdr.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
}

func (m *Manifest) UnmarshalJSON(b []byte) error {
	type Manifest struct {
		Architecture string `json:"architecture"`
		FsLayers     []struct {
			BlobSum string `json:"blobSum"`
		} `json:"fsLayers"`
		History []struct {
			V1Compatibility string `json:"v1Compatibility"`
		} `json:"history"`
		Name          string  `json:"name"`
		SchemaVersion float64 `json:"schemaVersion"`
		Signatures    []struct {
			Header struct {
				Alg string `json:"alg"`
				Jwk struct {
					Crv string `json:"crv"`
					Kid string `json:"kid"`
					Kty string `json:"kty"`
					X   string `json:"x"`
					Y   string `json:"y"`
				} `json:"jwk"`
			} `json:"header"`
			Protected string `json:"protected"`
			Signature string `json:"signature"`
		} `json:"signatures"`
		Tag string `json:"tag"`
	}

	manifest := &Manifest{}

	err := json.Unmarshal(b, &manifest)
	if err != nil {
		return fmt.Errorf("unmarshal %v", err)
	}

	m.blobs = make([]string, 0)
	for _, d := range manifest.FsLayers {
		m.blobs = append(m.blobs, d.BlobSum)
	}

	return nil
}

func (m *Manifest) ExtractHeaders(hdr *http.Header) {
	m.digest = hdr.Get("Docker-Content-Digest")
}

func manifest(conn *http.Client, url, image, tag string) (string, []string, error) {
	m := &Manifest{}

	err := get(conn, url+"/v2/"+image+"/manifests/"+tag, m)
	if err != nil {
		return "", nil, err
	}

	return m.digest, m.blobs, nil
}
