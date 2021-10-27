package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func connect(cmd *cobra.Command) (*http.Client, error) {
	url := cmd.Flag("registry").Value.String()

	// XXX construct client using TLS

	client := &http.Client{Transport: &http.Transport{}}

	resp, err := client.Get(url + "/v2/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		// error text in resp.Body
		return nil, errors.New("take action based on WWW-Authenticate")
	case http.StatusNotFound:
		return nil, errors.New("registry does not support v2 API")
	case http.StatusOK:
		break
	default:
		return nil, fmt.Errorf("bad status (%s/v2/) %d", url, resp.StatusCode)
	}

	// consume body so connection can be reused
	ioutil.ReadAll(resp.Body)

	ver := resp.Header.Get("Docker-Distribution-API-Version")
	if ver != "registry/2.0" {
		return nil, errors.New("registry does not support v2 API")
	}

	return client, nil
}

type Decoder interface {
	Method() string
	SetHeaders(*http.Header)
	UnmarshalJSON([]byte) error
	ExtractHeaders(*http.Header)
}

func get(conn *http.Client, url string, dec Decoder) error {
	req, err := http.NewRequest(dec.Method(), url, nil)
	if err != nil {
		return fmt.Errorf("new request: %v", err)
	}

	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json,application/vnd.docker.distribution.manifest.list.v2+json")

	resp, err := conn.Do(req)
	if err != nil {
		return fmt.Errorf("get: %v", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusAccepted:
	case http.StatusNotFound:
		//fmt.Println("Content-Type: ", resp.Header.Get("Content-Type"))
		//fmt.Println("Content-Length: ", resp.Header.Get("Content-Length"))
		if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
			if resp.Header.Get("Contenet-Length") == "None" {
				return fmt.Errorf("bad status: %s", http.StatusText(resp.StatusCode))
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("readall: %v", err)
			}

			var rpy struct {
				Errors []struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"errors"`
			}

			if err := json.Unmarshal(body, &rpy); err != nil {
				return fmt.Errorf("unmarshal not found: %v", err)
			}

			if len(rpy.Errors) == 1 {
				return fmt.Errorf("not found: code \"%s\" message \"%s\"", rpy.Errors[0].Code, rpy.Errors[0].Message)
			}
		}
		fallthrough
	default:
		return fmt.Errorf("bad status: %s", http.StatusText(resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("readall: %v", err)
	}

	dec.ExtractHeaders(&resp.Header)

	return dec.UnmarshalJSON(body)
}
