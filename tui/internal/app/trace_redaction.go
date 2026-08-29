package app

import "regexp"

var commonSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(authorization\s*:\s*(?:bearer\s+)?)[^\s]+`),
	regexp.MustCompile(`(?i)((?:api[_-]?key|apikey|token|secret|password)\s*[:=]\s*)[^\s,;]+`),
}

func redactCommonSecret(value string) string {
	out := value
	for _, pattern := range commonSecretPatterns {
		out = pattern.ReplaceAllString(out, `${1}***`)
	}
	return out
}
