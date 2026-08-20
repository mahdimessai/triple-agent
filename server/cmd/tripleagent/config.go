package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const defaultAddr = ":8080"

var defaultAllowedOrigins = []string{
	"http://localhost:3000",
	"http://127.0.0.1:3000",
}

type config struct {
	Addr           string
	AllowedOrigins []string
}

func loadConfig() (config, error) {
	addr := strings.TrimSpace(os.Getenv("TRIPLE_AGENT_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}

	origins, err := parseAllowedOrigins(os.Getenv("TRIPLE_AGENT_ALLOWED_ORIGINS"))
	if err != nil {
		return config{}, err
	}
	if len(origins) == 0 {
		origins = append([]string(nil), defaultAllowedOrigins...)
	}

	return config{Addr: addr, AllowedOrigins: origins}, nil
}

func parseAllowedOrigins(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	seen := make(map[string]struct{})
	origins := make([]string, 0)
	for _, value := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(value)
		if origin == "" {
			continue
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, fmt.Errorf("invalid allowed origin %q", origin)
		}
		normalized := strings.TrimSuffix(origin, "/")
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		origins = append(origins, normalized)
	}
	return origins, nil
}
