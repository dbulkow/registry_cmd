package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func check_registry(cmd *cobra.Command, args []string) {
	regvar := cmd.Flag("registry").Value.String()
	if regvar == "" {
		fmt.Fprintf(os.Stderr, "please set registry location")
		os.Exit(1)
	}
}

var RootCmd = &cobra.Command{
	Use:   "regcmd",
	Short: "Docker Registry CLI",
	Long: `Docker Registry command interface for maintenance.

Environment:

REGISTRY            Base URL for registry
REGISTRY_TLS_KEYS   Directory containing TLS keys
REGISTRY_TLS_VERIFY Enable TLS verification
`,
	PersistentPreRun: check_registry,
}

func main() {
	regvar := os.Getenv("REGISTRY")

	RootCmd.PersistentFlags().StringVar(&regvar, "registry", regvar, "Base URL for registry")

	RootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Display CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Verified with registry V2.4.1")
		},
	})

	// --tls - implied when any below are set

	// REGISTRY_TLS_KEYS=$(HOME)/.docker
	// --tlscacert $(REGISTRY_TLS_KEYS)/.docker/ca.pem
	// --tlscert   $(REGISTRY_TLS_KEYS)/.docker/cert.pem
	// --tlskey    $(REGISTRY_TLS_KEYS)/.docker/key.pem

	// REGISTRY_TLS_VERIFY=1
	// --tlsverify

	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}
