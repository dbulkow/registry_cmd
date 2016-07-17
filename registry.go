package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/dustin/go-humanize"
)

type Registry struct {
	Client  *http.Client
	BaseURL string
}

type Catalog struct {
	Repositories []string `json:"repositories"`
}

type Tags struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

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

func (r *Registry) VerifyV2() error {
	resp, err := r.Client.Get(r.BaseURL + "/v2/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		// error text in resp.Body
		return errors.New("take action based on WWW-Authenticate")
	case http.StatusNotFound:
		return errors.New("registry does not support v2 API")
	case http.StatusOK:
		break
	default:
		return errors.New(fmt.Sprintln("bad status (", r.BaseURL, "/v2/) ", resp.StatusCode))
	}

	// consume body so connection can be reused
	ioutil.ReadAll(resp.Body)

	ver := resp.Header.Get("Docker-Distribution-API-Version")
	if ver == "registry/2.0" {
		return nil
	}

	return errors.New("registry does not support v2 API")
}

func (r *Registry) Catalog() ([]string, error) {
	resp, err := r.Client.Get(r.BaseURL + "/v2/_catalog")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// decode error text if 4xx error code
		return nil, errors.New(fmt.Sprintln("bad status ", resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	catalog := new(Catalog)

	err = json.Unmarshal(body, &catalog)
	if err != nil {
		return nil, err
	}

	return catalog.Repositories, nil
}

func (r *Registry) Tags(image string) ([]string, error) {
	resp, err := r.Client.Get(r.BaseURL + "/v2/" + image + "/tags/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// decode error text if 4xx error code
		return nil, errors.New(fmt.Sprintln("bad status ", resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	tags := new(Tags)

	err = json.Unmarshal(body, &tags)
	if err != nil {
		return nil, err
	}

	return tags.Tags, nil
}

func (r *Registry) Manifest(image, tag string) ([]string, error) {
	resp, err := r.Client.Get(r.BaseURL + "/v2/" + image + "/manifests/" + tag)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// decode error text if 4xx error code
		return nil, errors.New(fmt.Sprintln("bad status ", resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	manifest := new(Manifest)

	err = json.Unmarshal(body, &manifest)
	if err != nil {
		return nil, err
	}

	blobs := make([]string, 0)
	for _, d := range manifest.FsLayers {
		blobs = append(blobs, d.BlobSum)
	}

	return blobs, nil
}

func (r *Registry) BlobSize(image, sum string) (int, error) {
	resp, err := r.Client.Head(r.BaseURL + "/v2/" + image + "/blobs/" + sum)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// decode error text if 4xx error code
		return 0, errors.New(fmt.Sprintln("bad status ", resp.StatusCode))
	}

	// consume body to connect can be reused
	ioutil.ReadAll(resp.Body)

	digest := resp.Header.Get("Docker-Content-Digest")
	if digest != sum {
		return 0, errors.New("digest mismatch")
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return 0, err
	}

	return size, nil
}

func (r *Registry) ImageSize(image, tag string) (int, error) {
	blobs, err := r.Manifest(image, tag)
	if err != nil {
		return 0, err
	}

	total := 0

	for _, b := range blobs {
		size, err := r.BlobSize(image, b)
		if err != nil {
			return 0, err
		}

		total += size
	}

	return total, nil
}

/*
 * There's no safety on this function.  If you want a blob deleted, it
 * will be removed from the server.
 */
func (r *Registry) DeleteBlob(image, sum string) error {
	// set header to obtain correct digest to delete:
	// Accept: application/vnd.docker.distribution.manifest.v2+json

	req, err := http.NewRequest("DELETE", r.BaseURL+"/v2/"+image+"/blobs/"+sum, nil)
	if err != nil {
		return err
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}

	// consume body to connect can be reused
	ioutil.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusAccepted:
		break
	case http.StatusNotFound:
		return errors.New("blob not found")
	default:
		return errors.New(fmt.Sprintln("bad status (", r.BaseURL, "/v2/) ", resp.StatusCode))
	}

	length := resp.Header.Get("Content-Length")
	if length != "None" {
		return errors.New("protocol error - content length should read None, got " + length)
	}

	return nil
}

type Item struct {
	Tags map[string][]string
}

func (r *Registry) DeleteImage(image, tag string) error {
	inventory := make(map[string]*Item, 0)

	images, err := r.Catalog()
	if err != nil {
		return err
	}

	for _, img := range images {
		inventory[img] = &Item{}

		tags, err := r.Tags(img)
		if err != nil {
			return err
		}

		inventory[img].Tags = make(map[string][]string, 1)
		for _, t := range tags {
			blobs, err := r.Manifest(img, t)
			if err != nil {
				return err
			}

			inventory[img].Tags[t] = blobs
		}
	}

	overlap := make([]string, 0)

	for _, img := range images {
		if img == image {
			continue
		}

		for t, blobs := range inventory[image].Tags {
			if t == tag {
				continue
			}

			for _, b := range blobs {
				for _, mb := range inventory[image].Tags[tag] {
					if b == mb {
						overlap = append(overlap, b)
					}
				}
			}
		}
	}

	remainder := make([]string, 0)
	for _, mb := range inventory[image].Tags[tag] {
		for _, b := range overlap {
			if b == mb {
				continue
			}
			remainder = append(remainder, b)
		}
	}

	for _, sum := range remainder {
		err := r.DeleteBlob(image, sum)
		if err != nil {
			return err
		}
	}

	// How to determine image:tag digest?
	digest := ""

	req, err := http.NewRequest("DELETE", r.BaseURL+"/v2/"+image+"/manifests/"+digest, nil)
	if err != nil {
		return err
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}

	// consume body to connect can be reused
	ioutil.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusAccepted:
		break
	case http.StatusNotFound:
		return errors.New("image not found")
	default:
		return errors.New(fmt.Sprintln("bad status (", r.BaseURL, "/v2/) ", resp.StatusCode))
	}

	length := resp.Header.Get("Content-Length")
	if length != "None" {
		return errors.New("protocol error - content length should read None, got " + length)
	}

	return nil
}

func main() {
	var (
		tls       = flag.Bool("tls", false, "Use TLS; implied by --tlsverify")
		tlscacert = flag.String("tlscacert", "~/.docker/ca.pem", "Trust certs signed only by this CA")
		tlscert   = flag.String("tlscert", "~/.docker/cert.pem", "Path to TLS certificate file")
		tlskey    = flag.String("tlskey", "~/.docker/key.pem", "Path to TLS key file")
		tlsverify = flag.Bool("tlsverify", false, "Use TLS and verify the remote")
	)

	registry := &Registry{
		Client:  &http.Client{Transport: &http.Transport{}},
		BaseURL: "http://yin.mno.stratus.com:5000",
	}

	err := registry.VerifyV2()
	if err != nil {
		fmt.Println(err)
		return
	}

	images, err := registry.Catalog()
	if err != nil {
		fmt.Println(err)
		return
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)

	for _, img := range images {
		tags, err := registry.Tags(img)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, tag := range tags {
			size, _ := registry.ImageSize(img, tag)
			fmt.Fprintf(w, "%s\t%s\t%8s\n", img, tag, humanize.Bytes(uint64(size)))
		}
	}
	w.Flush()
}
