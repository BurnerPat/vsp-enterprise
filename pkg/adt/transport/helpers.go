package transport

import (
	"net/http"
	"net/url"
	"strings"
)

// IsModifyingMethod returns true for HTTP methods that modify server state.
func IsModifyingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	default:
		return false
	}
}

// ExtractContextID extracts the sap-contextid value from a Set-Cookie header string.
func ExtractContextID(cookieHeader string) string {
	for _, part := range strings.Split(cookieHeader, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "sap-contextid=") {
			return strings.TrimPrefix(part, "sap-contextid=")
		}
	}
	// Also handle multiple cookies separated by comma.
	for _, cookie := range strings.Split(cookieHeader, ",") {
		for _, part := range strings.Split(cookie, ";") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "sap-contextid=") {
				return strings.TrimPrefix(part, "sap-contextid=")
			}
		}
	}
	return ""
}

// BuildURI constructs a URI path with query parameters, injecting sap-client and
// sap-language when configured. This logic is shared by all transports.
func BuildURI(path string, query url.Values, sapClient, sapLanguage string) (string, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	u, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	q := u.Query()
	if sapClient != "" {
		q.Set("sap-client", sapClient)
	}
	if sapLanguage != "" {
		q.Set("sap-language", sapLanguage)
	}
	for k, v := range query {
		for _, val := range v {
			q.Add(k, val)
		}
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// BuildFullURL constructs the full URL (base + path + query) for HTTP connections.
func BuildFullURL(baseURL, path string, query url.Values, sapClient, sapLanguage string) (string, error) {
	base := strings.TrimSuffix(baseURL, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	u, err := url.Parse(base + path)
	if err != nil {
		return "", err
	}

	q := u.Query()
	if sapClient != "" {
		q.Set("sap-client", sapClient)
	}
	if sapLanguage != "" {
		q.Set("sap-language", sapLanguage)
	}
	for k, v := range query {
		for _, val := range v {
			q.Add(k, val)
		}
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}
