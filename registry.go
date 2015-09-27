package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Catalog struct {
	Repositories []string `json:"repositories"`
}

type Tags struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func version(base string) (string, error) {
	resp, err := http.Get(base + "/v2/")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return "", errors.New("take action based on WWW-Authenticate")
	case http.StatusNotFound:
		return "", errors.New("V2 of the registry API not implemented")
	case http.StatusOK:
		break
	default:
		return "", errors.New(fmt.Sprintln("bad status (", base, "/v2/) ", resp.StatusCode))
	}

	return resp.Header.Get("Docker-Distribution-API-Version"), nil
}

func catalog(base string) ([]string, error) {
	resp, err := http.Get(base + "/v2/_catalog")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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

func tags(base, name string) ([]string, error) {
	resp, err := http.Get(base + "/v2/" + name + "/tags/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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
	base := "http://yin.mno.stratus.com:5000"

	ver, err := version(base)
	if err != nil {
		fmt.Println(err)
		return
	}

	if ver == "registry/v2.0" {
		fmt.Println("incorrect registry version")
		return
	}

	images, err := catalog(base)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, img := range images {
		tags, err := tags(base, img)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, t := range tags {
			fmt.Println(img + ":" + t)
		}
	}
}
