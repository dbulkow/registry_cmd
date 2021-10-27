package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	deleteCmd := &cobra.Command{
		Use:     "delete <image:tag>",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete an image",
		Run:     delete,
	}

	RootCmd.AddCommand(deleteCmd)
}

type Delete struct {
}

func (d *Delete) Method() string { return http.MethodDelete }

func (d *Delete) SetHeaders(hdr *http.Header) {}

func (d *Delete) UnmarshalJSON(b []byte) error {
	return nil
}

func (d *Delete) ExtractHeaders(hdr *http.Header) {}

func deleteImage(conn *http.Client, url, image, digest string) error {
	d := &Delete{}

	err := get(conn, url+"/v2/"+image+"/manifests/"+digest, d)
	if err != nil {
		return err
	}

	return nil
}

func delete(cmd *cobra.Command, args []string) {
	url := cmd.Flag("registry").Value.String()

	if len(args) != 1 {
		cmd.UsageFunc()(cmd)
		return
	}

	delimg := args[0]

	conn, err := connect(cmd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	images, err := catalog(conn, url)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for _, name := range images {
		tags, err := tags(conn, url, name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		for _, tag := range tags {
			if strings.Compare(delimg, name+":"+tag) == 0 {
				digest, _, err := manifest(conn, url, name, tag)
				if err != nil {
					fmt.Fprintf(os.Stderr, "manifest: %v\n", err)
					return
				}

				err = deleteImage(conn, url, name, digest)
				if err != nil {
					fmt.Fprintf(os.Stderr, "delete: %v\n", err)
					return
				}

				fmt.Printf("deleted %s:%s\n", name, tag)

				return
			}
		}
	}
}
