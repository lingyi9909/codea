package app

import "strings"

func redactCommonSecret(value string) string {
	lower := strings.ToLower(value)
	markers := []string{"api_key=", "api-key=", "apikey=", "password=", "token=", "secret=", "authorization:", "bearer ", "ghp_", "sk-"}
	for _, marker := range markers {
		searchFrom := 0
		for searchFrom < len(lower) {
			rel := strings.Index(lower[searchFrom:], marker)
			if rel < 0 {
				break
			}
			idx := searchFrom + rel
			start := idx + len(marker)
			end := start
			for end < len(value) && value[end] != ' ' && value[end] != '\t' && value[end] != '\n' && value[end] != '\r' && value[end] != ',' && value[end] != ';' {
				end++
			}
			value = value[:start] + "***" + value[end:]
			lower = strings.ToLower(value)
			searchFrom = start + len("***")
			if end == start {
				break
			}
		}
	}
	return value
}
