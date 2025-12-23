package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	ContentPath    string
	Host           string
	Port           int
	WatcherEnabled bool
	TemplatePath   string
}

func setupFlags() *Config {
	cfg := &Config{}

	// 1. Define Flags with default values or Environment Variable fallbacks
	flag.StringVar(&cfg.ContentPath, "path", getEnv("BLOG_PATH", ""), "Path to the folder containing README files (Required)")
	flag.IntVar(&cfg.Port, "port", getEnvInt("BLOG_PORT", 8080), "Port to host the blog on")
	flag.BoolVar(&cfg.WatcherEnabled, "watcher", getEnvBool("BLOG_WATCHER", true), "Enable hot reload on file changes")
	flag.StringVar(&cfg.TemplatePath, "template", getEnv("BLOG_TEMPLATE", "templates"), "Path to the template folder")
	flag.StringVar(&cfg.Host, "host", getEnv("BLOG_HOST", "127.0.0.1"), "Host (0.0.0.0) for public (defaults to localhost)")

	// Custom Usage/Help message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of Blog CLI:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables (overridden by flags):\n")
		fmt.Fprintf(os.Stderr, "  BLOG_PATH, BLOG_HOST, BLOG_PORT, BLOG_WATCHER, BLOG_TEMPLATE, BLOG_ASSETS\n")
	}

	flag.Parse()

	// 2. Validation: Ensure required fields are present
	if cfg.ContentPath == "" {
		fmt.Println("Error: The -path flag or BLOG_PATH env is required.")
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

// Helper functions to handle Environment Variables
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}
