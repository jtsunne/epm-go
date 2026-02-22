package main

import (
	"os"
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
		{
			name:     "URI with query string is stripped",
			uri:      "http://localhost:9200?x=1&y=2",
			wantBase: "http://localhost:9200",
		},
		{
			name:     "bare trailing question mark is stripped",
			uri:      "http://localhost:9200?",
			wantBase: "http://localhost:9200",
		},
		{
			name:     "URI with fragment is stripped",
			uri:      "http://localhost:9200#section",
			wantBase: "http://localhost:9200",
		},
		{
			name:     "URI with credentials and query string",
			uri:      "https://elastic:pw@host:9200?ssl=true",
			wantBase: "https://host:9200",
			wantUser: "elastic",
			wantPass: "pw",
		},
		// Port range validation
		{
			name:      "port zero",
			uri:       "http://localhost:0",
			wantError: true,
		},
		{
			name:      "port too high",
			uri:       "http://localhost:70000",
			wantError: true,
		},
		{
			name:      "port 65536",
			uri:       "http://localhost:65536",
			wantError: true,
		},
		{
			name:     "port 65535 accepted",
			uri:      "http://localhost:65535",
			wantBase: "http://localhost:65535",
		},
		// Plan-specified edge cases
		{
			name:     "plain http no credentials",
			uri:      "http://localhost:9200",
			wantBase: "http://localhost:9200",
			wantUser: "",
			wantPass: "",
		},
		{
			name:     "https with credentials and fqdn",
			uri:      "https://elastic:changeme@es.prod.example.com:9200",
			wantBase: "https://es.prod.example.com:9200",
			wantUser: "elastic",
			wantPass: "changeme",
		},
		{
			name:     "URL-encoded password p%40ss",
			uri:      "http://user:p%40ss@host:9200",
			wantBase: "http://host:9200",
			wantUser: "user",
			wantPass: "p@ss",
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

func TestCredentialResolution(t *testing.T) {
	tests := []struct {
		name     string
		uriUser  string
		uriPass  string
		envUser  string
		envPass  string
		flagUser string
		flagPass string
		wantUser string
		wantPass string
	}{
		{
			name:     "URI credentials used when nothing else set",
			uriUser:  "elastic",
			uriPass:  "changeme",
			wantUser: "elastic",
			wantPass: "changeme",
		},
		{
			name:     "env vars override URI credentials",
			uriUser:  "elastic",
			uriPass:  "changeme",
			envUser:  "envuser",
			envPass:  "envpass",
			wantUser: "envuser",
			wantPass: "envpass",
		},
		{
			name:     "flags override URI credentials",
			uriUser:  "elastic",
			uriPass:  "changeme",
			flagUser: "flaguser",
			flagPass: "flagpass",
			wantUser: "flaguser",
			wantPass: "flagpass",
		},
		{
			name:     "flags override env vars",
			envUser:  "envuser",
			envPass:  "envpass",
			flagUser: "flaguser",
			flagPass: "flagpass",
			wantUser: "flaguser",
			wantPass: "flagpass",
		},
		{
			name:     "priority chain: flag > env > URI",
			uriUser:  "uriuser",
			uriPass:  "uripass",
			envUser:  "envuser",
			envPass:  "envpass",
			flagUser: "flaguser",
			flagPass: "flagpass",
			wantUser: "flaguser",
			wantPass: "flagpass",
		},
		{
			name:     "only flag user set overrides URI user, URI pass used",
			uriUser:  "uriuser",
			uriPass:  "uripass",
			flagUser: "flaguser",
			wantUser: "flaguser",
			wantPass: "uripass",
		},
		{
			name:     "flag password with special chars including hash",
			uriUser:  "",
			uriPass:  "",
			flagUser: "root",
			flagPass: "op0107##",
			wantUser: "root",
			wantPass: "op0107##",
		},
		{
			name:     "env password with special chars including hash",
			envUser:  "root",
			envPass:  "op0107##",
			wantUser: "root",
			wantPass: "op0107##",
		},
		{
			name:     "empty strings at all sources",
			wantUser: "",
			wantPass: "",
		},
		{
			name:    "env user only, no URI or flag",
			envUser: "envonly",
			wantUser: "envonly",
			wantPass: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user, pass := resolveCredentials(tc.uriUser, tc.uriPass, tc.envUser, tc.envPass, tc.flagUser, tc.flagPass)
			if user != tc.wantUser {
				t.Errorf("user = %q, want %q", user, tc.wantUser)
			}
			if pass != tc.wantPass {
				t.Errorf("pass = %q, want %q", pass, tc.wantPass)
			}
		})
	}
}

// TestCredentialResolutionEnvVars tests that resolveCredentials integrates
// correctly with real environment variables via os.Getenv.
func TestCredentialResolutionEnvVars(t *testing.T) {
	t.Run("ES_USER and ES_PASSWORD override URI", func(t *testing.T) {
		t.Setenv("ES_USER", "envuser")
		t.Setenv("ES_PASSWORD", "envpass")

		user, pass := resolveCredentials("uriuser", "uripass", os.Getenv("ES_USER"), os.Getenv("ES_PASSWORD"), "", "")
		if user != "envuser" {
			t.Errorf("user = %q, want %q", user, "envuser")
		}
		if pass != "envpass" {
			t.Errorf("pass = %q, want %q", pass, "envpass")
		}
	})

	t.Run("flag overrides ES_PASSWORD env var", func(t *testing.T) {
		t.Setenv("ES_USER", "envuser")
		t.Setenv("ES_PASSWORD", "envpass")

		user, pass := resolveCredentials("", "", os.Getenv("ES_USER"), os.Getenv("ES_PASSWORD"), "flaguser", "flagpass")
		if user != "flaguser" {
			t.Errorf("user = %q, want %q", user, "flaguser")
		}
		if pass != "flagpass" {
			t.Errorf("pass = %q, want %q", pass, "flagpass")
		}
	})

	t.Run("ES_PASSWORD with hash character", func(t *testing.T) {
		t.Setenv("ES_USER", "root")
		t.Setenv("ES_PASSWORD", "op0107##")

		user, pass := resolveCredentials("", "", os.Getenv("ES_USER"), os.Getenv("ES_PASSWORD"), "", "")
		if user != "root" {
			t.Errorf("user = %q, want %q", user, "root")
		}
		if pass != "op0107##" {
			t.Errorf("pass = %q, want %q", pass, "op0107##")
		}
	})
}
