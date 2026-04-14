package headers

import (
	"bytes"
	"fmt"
	"strings"
)

type Headers map[string]string

func NewHeaders() Headers {
	return Headers{}
}

func (headers Headers) Parse(data []byte) (n int, done bool, err error) {
	lineEnd := bytes.Index(data, []byte("\r\n"))
	if lineEnd == -1 {
		return 0, false, nil
	}

	if lineEnd == 0 {
		return 2, true, nil
	}

	line := string(data[:lineEnd])
	colonIndex := strings.Index(line, ":")
	if colonIndex == -1 {
		return 0, false, fmt.Errorf("invalid header: missing colon")
	}

	key := line[:colonIndex]
	if key != strings.TrimSpace(key) {
		return 0, false, fmt.Errorf("invalid header: spacing before key")
	}

	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return 0, false, fmt.Errorf("invalid header: empty key")
	}
	if strings.ContainsAny(trimmedKey, " \t") {
		return 0, false, fmt.Errorf("invalid header: spacing in key")
	}
	if !isValidFieldName(trimmedKey) {
		return 0, false, fmt.Errorf("invalid header: invalid key character")
	}

	value := strings.TrimSpace(line[colonIndex+1:])
	normalizedKey := strings.ToLower(trimmedKey)
	if existingValue, exists := headers[normalizedKey]; exists {
		headers[normalizedKey] = existingValue + ", " + value
	} else {
		headers[normalizedKey] = value
	}

	return lineEnd + 2, false, nil
}

func (headers Headers) Get(key string) string {
	return headers[strings.ToLower(key)]
}

func (headers Headers) Set(key, value string) {
	headers[strings.ToLower(key)] = value
}

func (headers Headers) Delete(key string) {
	delete(headers, strings.ToLower(key))
}

func isValidFieldName(name string) bool {
	if len(name) == 0 {
		return false
	}

	for i := 0; i < len(name); i++ {
		c := name[i]
		isAlpha := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
		isDigit := c >= '0' && c <= '9'
		isAllowedSymbol := strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c))
		if !isAlpha && !isDigit && !isAllowedSymbol {
			return false
		}
	}

	return true
}
