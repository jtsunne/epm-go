package main

import (
	"testing"
)

func TestParseESURI(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		wantBase  string
		wantUser  string
		wantPass  string
		wantError bool
	}{
		{
			name:     "plain http URI",
			uri:      "http://localhost:9200",
			wantBase: "http://localhost:9200",
		},
		{
			name:     "plain https URI",
			uri:      "https://es.example.com:9200",
			wantBase: "https://es.example.com:9200",
		},
		{
			name:     "URI with credentials",
			uri:      "http://elastic:changeme@localhost:9200",
			wantBase: "http://localhost:9200",
			wantUser: "elastic",
			wantPass: "changeme",
		},
		{
			name:     "URI with special chars in password",
			uri:      "https://user:p%40ss%3Aword@host:9200",
			wantBase: "https://host:9200",
			wantUser: "user",
			wantPass: "p@ss:word",
		},
		{
			name:      "no scheme",
			uri:       "localhost:9200",
			wantError: true,
		},
		{
			name:      "unsupported scheme",
			uri:       "ftp://localhost:9200",
			wantError: true,
		},
		{
			name:      "empty URI",
			uri:       "",
			wantError: true,
		},
		{
			name:      "hostless URI",
			uri:       "http://",
			wantError: true,
		},
		{
			name:      "port-only authority",
			uri:       "http://:9200",
			wantError: true,
		},
		{
			name:     "password-only userinfo",
			uri:      "http://:secret@localhost:9200",
			wantBase: "http://localhost:9200",
			wantUser: "",
			wantPass: "secret",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			base, user, pass, err := parseESURI(tc.uri)
			if tc.wantError {
				if err == nil {
					t.Errorf("parseESURI(%q): expected error, got nil", tc.uri)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseESURI(%q): unexpected error: %v", tc.uri, err)
			}
			if base != tc.wantBase {
				t.Errorf("baseURL = %q, want %q", base, tc.wantBase)
			}
			if user != tc.wantUser {
				t.Errorf("username = %q, want %q", user, tc.wantUser)
			}
			if pass != tc.wantPass {
				t.Errorf("password = %q, want %q", pass, tc.wantPass)
			}
		})
	}
}
