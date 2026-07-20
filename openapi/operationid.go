package openapi

import (
	"slices"
	"strings"
)

func generateOperationID(method, pattern string) string {
	segments := nonParamSegments(pattern)
	if len(segments) == 0 {
		return strings.ToLower(method)
	}

	endsWithParam := endsOnParam(pattern)

	verb, singLast := verbFor(method, endsWithParam)

	out := make([]string, len(segments))
	copy(out, segments)

	if singLast {
		out[len(out)-1] = singularize(out[len(out)-1])
	}

	return verb + "-" + strings.Join(out, "-")
}

func nonParamSegments(pattern string) []string {
	parts := strings.Split(pattern, "/")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			continue
		}
		if isVersionSegment(p) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func endsOnParam(pattern string) bool {
	parts := strings.Split(strings.TrimRight(pattern, "/"), "/")
	for _, part := range slices.Backward(parts) {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		return strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}")
	}
	return false
}

func verbFor(method string, endsWithParam bool) (verb string, singularizeLast bool) {
	switch strings.ToUpper(method) {
	case "GET":
		if endsWithParam {
			return "get", true
		}
		return "list", false
	case "POST":
		return "create", true
	case "PUT":
		return "replace", true
	case "PATCH":
		return "update", true
	case "DELETE":
		return "delete", true
	case "HEAD":
		return "head", false
	case "OPTIONS":
		return "options", false
	default:
		return strings.ToLower(method), false
	}
}

func singularize(s string) string {
	switch {
	case strings.HasSuffix(s, "ies"):
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(s, "ses") || strings.HasSuffix(s, "xes") ||
		strings.HasSuffix(s, "zes") || strings.HasSuffix(s, "ches") ||
		strings.HasSuffix(s, "shes"):
		return s[:len(s)-2] // remove "es"
	case strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss"):
		return s[:len(s)-1]
	}
	return s
}

func isVersionSegment(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for _, c := range s[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
