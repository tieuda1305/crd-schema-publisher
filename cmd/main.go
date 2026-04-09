package main

import (
	"fmt"
	"os"

	"github.com/sholdee/crd-schema-publisher/extractor"
	"github.com/sholdee/crd-schema-publisher/index"
	"github.com/sholdee/crd-schema-publisher/publisher"
)

func main() {
	cmd := "run"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "run":
		if err := runAll(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "extract":
		if err := runExtract(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "upload":
		if err := runUpload(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\nusage: crd-schema-publisher [run|extract|upload]\n", cmd)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}

func runExtract() error {
	outputDir := getEnv("OUTPUT_DIR", "/output")
	kubeContext := os.Getenv("KUBECTL_CONTEXT")

	fmt.Println("Building Kubernetes client...")
	client, err := extractor.BuildClient(kubeContext)
	if err != nil {
		return fmt.Errorf("building client: %w", err)
	}

	fmt.Println("Listing CRDs...")
	crds, err := extractor.ListCRDs(client)
	if err != nil {
		return err
	}
	fmt.Printf("Found %d CRDs\n", len(crds))

	if len(crds) == 0 {
		fmt.Println("No CRDs found, nothing to extract")
		return nil
	}

	count, err := extractor.WriteSchemas(crds, outputDir)
	if err != nil {
		return err
	}
	fmt.Printf("Wrote %d JSON schemas to %s\n", count, outputDir)

	fmt.Println("Generating index.html...")
	if err := index.Generate(outputDir); err != nil {
		return fmt.Errorf("generating index: %w", err)
	}

	fmt.Println("Extract complete")
	return nil
}

func runUpload() error {
	outputDir := getEnv("OUTPUT_DIR", "/output")

	apiToken, err := requireEnv("CLOUDFLARE_API_TOKEN")
	if err != nil {
		return err
	}
	accountID, err := requireEnv("CLOUDFLARE_ACCOUNT_ID")
	if err != nil {
		return err
	}
	projectName := getEnv("CF_PAGES_PROJECT", "kubernetes-schemas")

	p := &publisher.Publisher{
		APIToken:    apiToken,
		AccountID:   accountID,
		ProjectName: projectName,
	}

	return p.Publish(outputDir)
}

func runAll() error {
	if err := runExtract(); err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	if err := runUpload(); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	return nil
}
