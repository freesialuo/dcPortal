package handler

import (
	"fmt"
	"net/url"
	"strings"
)

func normalizeScopes(raw string) string {
	scopes := strings.Fields(strings.TrimSpace(raw))
	if len(scopes) == 0 {
		return "bot"
	}
	return strings.Join(scopes, " ")
}

func validatePermissions(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return fmt.Errorf("permissions must be a numeric string")
		}
	}
	return nil
}

func validateRedirectURI(raw string) error {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed == nil {
		return fmt.Errorf("redirect URI is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("redirect URI must use http or https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return fmt.Errorf("redirect URI must include host")
	}
	return nil
}
