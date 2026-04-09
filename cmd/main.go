package main

import (
	"fmt"
	"os"
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
	fmt.Println("extract: not yet implemented")
	return nil
}

func runUpload() error {
	fmt.Println("upload: not yet implemented")
	return nil
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
