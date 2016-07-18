package main

import (
	"fmt"
	"os"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

var getsize, getdigest, usebytes bool

func init() {
	listCmd := &cobra.Command{
		Use:     "list [glob]",
		Aliases: []string{"ls"},
		Short:   "List images in the registry",
		Run:     list,
	}

	listCmd.Flags().BoolVarP(&getsize, "size", "s", false, "List image size")
	listCmd.Flags().BoolVarP(&getdigest, "digest", "d", false, "List image size")
	listCmd.Flags().BoolVarP(&usebytes, "bytes", "b", false, "Display sizes in bytes")

	RootCmd.AddCommand(listCmd)
}

func list(cmd *cobra.Command, args []string) {
	url := cmd.Flag("registry").Value.String()

	var filter string
	if len(args) > 0 {
		filter = args[0]
	}

	conn, err := connect(cmd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	type Image struct {
		name string
		tags []string
	}

	imageset := make([]*Image, 0)

	images, err := catalog(conn, url)

	n := 0
	for _, name := range images {
		if filter != "" && !Glob(filter, name) {
			continue
		}

		tags, err := tags(conn, url, name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		img := &Image{
			name: name,
			tags: tags,
		}

		for _, t := range tags {
			if n < len(name+t) {
				n = len(name + t)
			}
		}

		imageset = append(imageset, img)
	}

	for _, i := range imageset {
		for _, t := range i.tags {
			fmt.Printf("%-*s", n+1, i.name+":"+t)

			if getsize {
				sz, err := imagesize(conn, url, i.name, t)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				if usebytes {
					fmt.Printf(" %12d", sz)
				} else {
					fmt.Printf(" %12s", humanize.Bytes(sz))
				}
			}

			if getdigest {
				digest, _, err := manifest(conn, url, i.name, t)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				fmt.Printf(" %s", digest)
			}

			fmt.Printf("\n")
		}
	}
}
