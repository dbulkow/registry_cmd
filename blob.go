package main

import (
	"errors"
	"net/http"
	"strconv"
)

type Blob struct {
	digest string
	length string
}

func (b *Blob) Method() string { return http.MethodGet }

func (b *Blob) SetHeaders(hdr *http.Header) {}

func (b *Blob) UnmarshalJSON(body []byte) error {
	return nil
}

func (b *Blob) ExtractHeaders(hdr *http.Header) {
	b.digest = hdr.Get("Docker-Content-Digest")
	b.length = hdr.Get("Content-Length")
}

func blobsize(conn *http.Client, url, image, blob string) (uint64, error) {
	b := &Blob{}

	err := get(conn, url+"/v2/"+image+"/blobs/"+blob, b)
	if err != nil {
		return 0, err
	}

	if b.digest != blob {
		return 0, errors.New("digest mismatch")
	}

	size, err := strconv.ParseUint(b.length, 10, 64)
	if err != nil {
		return 0, err
	}

	return size, nil
}

func imagesize(conn *http.Client, url, image, tag string) (uint64, error) {
	_, blobs, err := manifest(conn, url, image, tag)
	if err != nil {
		return 0, err
	}

	var total uint64

	for _, b := range blobs {
		size, err := blobsize(conn, url, image, b)
		if err != nil {
			return 0, err
		}

		total += size
	}

	return total, nil
}
