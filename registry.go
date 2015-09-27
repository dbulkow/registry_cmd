package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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

func (r *Registry) Tags(img string) ([]string, error) {
	resp, err := r.Client.Get(r.BaseURL + "/v2/" + img + "/tags/list")
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

func main() {
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

	for _, img := range images {
		tags, err := registry.Tags(img)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, t := range tags {
			fmt.Println(img + ":" + t)
		}
	}
}
