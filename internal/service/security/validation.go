package security

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

func (r ValidationResult) Error() string {
	if !r.Valid {
		var errs []string
		for _, e := range r.Errors {
			errs = append(errs, e.Error())
		}
		return strings.Join(errs, "; ")
	}
	return ""
}

func ValidateEmail(email string) error {
	const op = "security.ValidateEmail"

	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("%s: email is required", op)
	}

	if len(email) > 254 {
		return fmt.Errorf("%s: email too long", op)
	}

	if !emailRegex.MatchString(email) {
		return fmt.Errorf("%s: invalid email format", op)
	}

	return nil
}

func ValidatePhone(phone string) error {
	const op = "security.ValidatePhone"

	phone = strings.TrimSpace(phone)
	if phone == "" {
		return fmt.Errorf("%s: phone is required", op)
	}

	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("%s: invalid phone format", op)
	}

	return nil
}

func ValidateUUID(id string) error {
	const op = "security.ValidateUUID"

	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%s: UUID is required", op)
	}

	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("%s: invalid UUID format", op)
	}

	return nil
}

func ValidateUsername(username string) error {
	const op = "security.ValidateUsername"

	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("%s: username is required", op)
	}

	if len(username) < 3 || len(username) > 32 {
		return fmt.Errorf("%s: username must be 3-32 characters", op)
	}

	validUsername := regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)
	if !validUsername.MatchString(username) {
		return fmt.Errorf("%s: username contains invalid characters", op)
	}

	return nil
}

func SanitizeString(input string) string {
	input = strings.TrimSpace(input)

	var builder strings.Builder
	for _, r := range input {
		if r == '<' || r == '>' || r == '&' || r == '"' || r == '\'' {
			continue
		}
		if unicode.IsPrint(r) {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func ValidatePassword(password string) error {
	const op = "security.ValidatePassword"

	if len(password) < 8 {
		return fmt.Errorf("%s: password must be at least 8 characters", op)
	}

	if len(password) > 128 {
		return fmt.Errorf("%s: password too long", op)
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		if unicode.IsUpper(c) {
			hasUpper = true
		}
		if unicode.IsLower(c) {
			hasLower = true
		}
		if unicode.IsDigit(c) {
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("%s: password must contain uppercase, lowercase, and digit", op)
	}

	return nil
}

func ValidateRedirectURI(uri string) error {
	const op = "security.ValidateRedirectURI"

	uri = strings.TrimSpace(uri)
	if uri == "" {
		return fmt.Errorf("%s: redirect URI is required", op)
	}

	parsed, err := regexp.Compile(`^https?://[a-zA-Z0-9\-\.]+(\.[a-zA-Z]{2,})(:[0-9]+)?(/.*)?$`)
	if err != nil {
		return fmt.Errorf("%s: invalid URI pattern", op)
	}

	if !parsed.MatchString(uri) {
		return fmt.Errorf("%s: invalid redirect URI format", op)
	}

	return nil
}

func ValidateScope(scope string) error {
	const op = "security.ValidateScope"

	scope = strings.TrimSpace(scope)
	if scope == "" {
		return fmt.Errorf("%s: scope is required", op)
	}

	validScopes := map[string]bool{
		"openid": true, "profile": true, "email": true,
		"roles": true, "offline_access": true,
	}

	scopes := strings.Split(scope, " ")
	for _, s := range scopes {
		if !validScopes[s] {
			return fmt.Errorf("%s: invalid scope: %s", op, s)
		}
	}

	return nil
}