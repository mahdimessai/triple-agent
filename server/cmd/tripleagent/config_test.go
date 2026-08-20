package main

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("TRIPLE_AGENT_ADDR", "")
	t.Setenv("TRIPLE_AGENT_ALLOWED_ORIGINS", "")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != defaultAddr {
		t.Fatalf("addr=%q", cfg.Addr)
	}
	if len(cfg.AllowedOrigins) != len(defaultAllowedOrigins) {
		t.Fatalf("origins=%v", cfg.AllowedOrigins)
	}
}

func TestLoadConfigNormalizesOrigins(t *testing.T) {
	t.Setenv("TRIPLE_AGENT_ADDR", "127.0.0.1:9999")
	t.Setenv("TRIPLE_AGENT_ALLOWED_ORIGINS", " https://game.example/,https://game.example,http://localhost:3000 ")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "127.0.0.1:9999" {
		t.Fatalf("addr=%q", cfg.Addr)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[0] != "https://game.example" || cfg.AllowedOrigins[1] != "http://localhost:3000" {
		t.Fatalf("origins=%v", cfg.AllowedOrigins)
	}
}

func TestLoadConfigRejectsMalformedOrigin(t *testing.T) {
	t.Setenv("TRIPLE_AGENT_ALLOWED_ORIGINS", "not-an-origin")
	if _, err := loadConfig(); err == nil {
		t.Fatal("malformed origin was accepted")
	}
}
