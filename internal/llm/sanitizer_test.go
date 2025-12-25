package llm

import (
	"strings"
	"testing"
)

// NOTE: All "secrets" in this file are INTENTIONALLY FAKE test patterns.
// They are designed to test the sanitizer and are NOT real credentials.

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
	// Using obviously fake bearer tokens for testing
	tests := []struct {
		input string
		want  string
	}{
		{"Authorization: Bearer FAKE-test-token.for.testing", "Authorization: Bearer [REDACTED_TOKEN]"},
		{"bearer FAKE-access-token-12345", "Bearer [REDACTED_TOKEN]"},
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
		// Using AWS example keys that are clearly fake but match regex format
		{"AWS Access Key", "AKIAFAKEEXAMPLEKEY12", "[REDACTED_AWS_KEY]"},
		{"AWS Secret", "aws_secret_access_key=FAKEexampleSECRETkeyVALUE123456789abcdef", "aws_secret_access_key=[REDACTED_AWS_SECRET]"},
		{"Asia prefix", "ASIAFAKEACCESSKEY123", "[REDACTED_AWS_KEY]"},
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
	// Using obviously fake private key for testing
	input := `-----BEGIN PRIVATE KEY-----
FAKE-TEST-KEY-NOT-REAL-AAAABBBBCCCCDDDD1111222233334444
-----END PRIVATE KEY-----`

	sanitizer := DefaultSanitizer()
	got := sanitizer.Sanitize(input)

	if got != "[REDACTED_PRIVATE_KEY]" {
		t.Errorf("Expected private key to be redacted, got: %s", got)
	}
}

func TestSanitize_RSAPrivateKey(t *testing.T) {
	// Using obviously fake RSA key for testing
	input := `-----BEGIN RSA PRIVATE KEY-----
FAKE-RSA-TEST-KEY-NOT-REAL-AAAABBBBCCCC
-----END RSA PRIVATE KEY-----`

	sanitizer := DefaultSanitizer()
	got := sanitizer.Sanitize(input)

	if got != "[REDACTED_PRIVATE_KEY]" {
		t.Errorf("Expected RSA private key to be redacted, got: %s", got)
	}
}

func TestSanitize_GitHubToken(t *testing.T) {
	// Using fake GitHub PAT pattern for testing (ghp_ + 36 alphanumeric chars)
	input := "ghp_FAKETOKENabcdefghij1234567890abcdefX"
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
		// Using obviously fake database passwords for testing
		{"PostgreSQL", "postgresql://user:FAKEPASS@localhost:5432/db", "postgresql://[user]:[REDACTED]@localhost:5432/db"},
		{"MongoDB", "mongodb://admin:FAKEPASS@cluster.mongodb.net/mydb", "mongodb://[user]:[REDACTED]@cluster.mongodb.net/mydb"},
		{"MySQL", "mysql://root:FAKEPASS@127.0.0.1:3306/app", "mysql://[user]:[REDACTED]@127.0.0.1:3306/app"},
		{"Redis", "redis://default:FAKEPASS@redis.io:6379", "redis://[user]:[REDACTED]@redis.io:6379"},
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
	// Using obviously fake JWT pattern for testing (not a valid JWT)
	input := "eyJGQUtFIjoiVEVTVCJ9.eyJGQUtFIjoiREFUQSJ9.FAKESIGNATURE"
	sanitizer := DefaultSanitizer()
	got := sanitizer.Sanitize(input)

	if !strings.Contains(got, "[REDACTED_JWT]") {
		t.Errorf("Expected JWT to be redacted, got: %s", got)
	}
}

func TestSanitize_SlackToken(t *testing.T) {
	tests := []struct {
		input string
	}{
		// Using clearly fake placeholder values to avoid secret scanning
		{"xoxb-FAKE-PLACEHOLDER-TOKEN"},
		{"xoxp-FAKE-TEST-VALUE"},
		{"xoxa-FAKE-TOKEN"},
	}

	sanitizer := DefaultSanitizer()
	for _, tt := range tests {
		got := sanitizer.Sanitize(tt.input)
		if !strings.Contains(got, "[REDACTED_SLACK_TOKEN]") {
			t.Errorf("Expected Slack token to be redacted, got: %s", got)
		}
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
