package llm

import (
	"strings"
	"testing"
)

func TestSanitize_APIKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"api_key with equals", "api_key=sk-abc123def456ghi789jkl", "api_key=[REDACTED_API_KEY]"},
		{"apikey no separator", "apikey=abcdefghijklmnopqrstuvwxyz", "apikey=[REDACTED_API_KEY]"},
		{"API-KEY quoted", `API-KEY="my_super_secret_key_123456"`, `API-KEY=[REDACTED_API_KEY]`},
		{"mixed case", "Api_Key:verylongsecretkeyvalue123456", "Api_Key=[REDACTED_API_KEY]"},
	}

	sanitizer := DefaultSanitizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizer.Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitize_BearerToken(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.token", "Authorization: Bearer [REDACTED_TOKEN]"},
		{"bearer my-access-token-12345", "Bearer [REDACTED_TOKEN]"},
	}

	sanitizer := DefaultSanitizer()
	for _, tt := range tests {
		got := sanitizer.Sanitize(tt.input)
		if got != tt.want {
			t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitize_AWSCredentials(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"AWS Access Key", "AKIAIOSFODNN7EXAMPLE", "[REDACTED_AWS_KEY]"},
		{"AWS Secret", "aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "aws_secret_access_key=[REDACTED_AWS_SECRET]"},

		{"Asia prefix", "ASIAACCESSKEYID12345", "[REDACTED_AWS_KEY]"},
	}

	sanitizer := DefaultSanitizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizer.Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitize_PrivateKey(t *testing.T) {
	input := `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC7JHoJfg6yNzLM
-----END PRIVATE KEY-----`

	sanitizer := DefaultSanitizer()
	got := sanitizer.Sanitize(input)

	if got != "[REDACTED_PRIVATE_KEY]" {
		t.Errorf("Expected private key to be redacted, got: %s", got)
	}
}

func TestSanitize_RSAPrivateKey(t *testing.T) {
	input := `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy
-----END RSA PRIVATE KEY-----`

	sanitizer := DefaultSanitizer()
	got := sanitizer.Sanitize(input)

	if got != "[REDACTED_PRIVATE_KEY]" {
		t.Errorf("Expected RSA private key to be redacted, got: %s", got)
	}
}

func TestSanitize_GitHubToken(t *testing.T) {

	input := "ghp_abcdefghijklmnopqrstuvwxyz1234567890"
	sanitizer := DefaultSanitizer()
	got := sanitizer.Sanitize(input)

	if !strings.Contains(got, "[REDACTED_GITHUB_TOKEN]") {
		t.Errorf("Expected GitHub token to be redacted, got: %s", got)
	}
}

func TestSanitize_DatabaseURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"PostgreSQL", "postgresql://user:secretpass@localhost:5432/db", "postgresql://[user]:[REDACTED]@localhost:5432/db"},
		{"MongoDB", "mongodb://admin:password123@cluster.mongodb.net/mydb", "mongodb://[user]:[REDACTED]@cluster.mongodb.net/mydb"},
		{"MySQL", "mysql://root:topsecret@127.0.0.1:3306/app", "mysql://[user]:[REDACTED]@127.0.0.1:3306/app"},
		{"Redis", "redis://default:myredispassword@redis.io:6379", "redis://[user]:[REDACTED]@redis.io:6379"},
	}

	sanitizer := DefaultSanitizer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizer.Sanitize(tt.input)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitize_JWTToken(t *testing.T) {

	input := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	sanitizer := DefaultSanitizer()
	got := sanitizer.Sanitize(input)

	if !strings.Contains(got, "[REDACTED_JWT]") {
		t.Errorf("Expected JWT to be redacted, got: %s", got)
	}
}

func TestSanitize_Password(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"password=mysecret123", "password=[REDACTED]"},
		{"secret:verysecretvalue", "secret=[REDACTED]"},
		{"TOKEN=abc123xyz789", "TOKEN=[REDACTED]"},
	}

	sanitizer := DefaultSanitizer()
	for _, tt := range tests {
		got := sanitizer.Sanitize(tt.input)
		if got != tt.want {
			t.Errorf("Sanitize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeWithReport(t *testing.T) {
	input := "api_key=supersecretapikey12345 and password=secret123"
	sanitizer := DefaultSanitizer()

	sanitized, found := sanitizer.SanitizeWithReport(input)

	if len(found) == 0 {
		t.Error("Expected to find secrets in report")
	}
	if strings.Contains(sanitized, "supersecretapikey12345") {
		t.Error("API key should be redacted")
	}
	if strings.Contains(sanitized, "secret123") {
		t.Error("Password should be redacted")
	}
}

func TestContainsSecrets(t *testing.T) {
	sanitizer := DefaultSanitizer()

	tests := []struct {
		input string
		want  bool
	}{
		{"api_key=secretkey12345678901234", true},
		{"Bearer mytoken123", true},
		{"Just some normal text", false},
		{"password=secretpassword", true},
		{"public data only", false},
	}

	for _, tt := range tests {
		got := sanitizer.ContainsSecrets(tt.input)
		if got != tt.want {
			t.Errorf("ContainsSecrets(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestTruncateForLLM(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		check  func(string) bool
	}{
		{
			name:   "short input unchanged",
			input:  "short",
			maxLen: 100,
			check:  func(s string) bool { return s == "short" },
		},
		{
			name:   "long input truncated",
			input:  strings.Repeat("a", 1000),
			maxLen: 100,
			check:  func(s string) bool { return len(s) <= 100 && strings.Contains(s, "[truncated]") },
		},
		{
			name:   "preserves start and end",
			input:  "START" + strings.Repeat("x", 500) + "END",
			maxLen: 100,
			check:  func(s string) bool { return strings.HasPrefix(s, "START") && strings.HasSuffix(s, "END") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateForLLM(tt.input, tt.maxLen)
			if !tt.check(got) {
				t.Errorf("TruncateForLLM failed check: got %q (len=%d)", got, len(got))
			}
		})
	}
}

func TestPrepareForLLM(t *testing.T) {

	input := "  api_key=secretkey12345678901234 " + strings.Repeat("x", 200) + "  "

	got := PrepareForLLM(input, 100)

	if strings.Contains(got, "secretkey12345678901234") {
		t.Error("API key should be redacted")
	}

	if len(got) > 100 {
		t.Errorf("Output should be <= 100 chars, got %d", len(got))
	}

	if strings.HasPrefix(got, " ") || strings.HasSuffix(got, " ") {
		t.Error("Output should be trimmed")
	}
}

func TestMaskEnvVars(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"export API_KEY=mysecret", "export API_KEY=[REDACTED]"},
		{"SECRET=topsecret", "SECRET=[REDACTED]"},
		{"PASSWORD=pass123", "PASSWORD=[REDACTED]"},
	}

	for _, tt := range tests {
		got := MaskEnvVars(tt.input)
		if got != tt.want {
			t.Errorf("MaskEnvVars(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAddPattern(t *testing.T) {
	sanitizer := DefaultSanitizer()
	initialCount := sanitizer.PatternCount()

	err := sanitizer.AddPattern("Custom", `custom-secret-\d+`, "[REDACTED_CUSTOM]")
	if err != nil {
		t.Fatalf("AddPattern failed: %v", err)
	}

	if sanitizer.PatternCount() != initialCount+1 {
		t.Error("Pattern count should increase by 1")
	}

	got := sanitizer.Sanitize("Found custom-secret-12345 in logs")
	if !strings.Contains(got, "[REDACTED_CUSTOM]") {
		t.Errorf("Custom pattern should match, got: %s", got)
	}
}

func TestAddPattern_InvalidRegex(t *testing.T) {
	sanitizer := DefaultSanitizer()

	err := sanitizer.AddPattern("Invalid", "[invalid", "replacement")
	if err == nil {
		t.Error("Expected error for invalid regex")
	}
}

func TestSanitizeOutput(t *testing.T) {
	input := "api_key=secretkey12345678901234 export SECRET=mysecret"

	got := SanitizeOutput(input)

	if strings.Contains(got, "secretkey12345678901234") {
		t.Error("API key should be redacted")
	}
	if strings.Contains(got, "mysecret") {
		t.Error("Exported secret should be redacted")
	}
}

func TestSanitizeForLLM(t *testing.T) {
	input := "password=mysecret123"
	got := SanitizeForLLM(input)

	if strings.Contains(got, "mysecret123") {
		t.Error("Password should be redacted by global sanitizer")
	}
}
