package config

import (
	"reflect"
	"testing"
)

func TestParseOrigins(t *testing.T) {
	got := parseOrigins(" https://dashboard.example.com, http://localhost:3000 ,,")
	want := []string{"https://dashboard.example.com", "http://localhost:3000"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseOrigins = %q, want %q", got, want)
	}
}

func TestLoadReadsAllowedOrigins(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://dashboard.example.com")
	if got := Load().AllowedOrigins; !reflect.DeepEqual(got, []string{"https://dashboard.example.com"}) {
		t.Fatalf("AllowedOrigins = %q", got)
	}
}
