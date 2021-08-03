package utils

import (
	"regexp"
	"strings"
)

// SanitizeUrl removes excess forward slashes as well as pad the end of the URL with / if suffixSlash is true
func SanitizeUrl(url string, suffixSlash bool) string {
	if len(url) == 0 {
		return ""
	}
	strBuilder := strings.Builder{}
	isAtForwardSlash := false
	startLoc := 0
	protocolRegExp, _ := regexp.Compile("https?://")
	protocolMatch := protocolRegExp.FindAllString(url, 1)
	if len(protocolMatch) > 0 {
		startLoc += len(protocolMatch[0])
		strBuilder.WriteString(protocolMatch[0])
	}

	for _, c := range url[startLoc:] {
		if isAtForwardSlash && byte(c) == '/' {
			continue
		}
		isAtForwardSlash = byte(c) == '/'
		strBuilder.WriteByte(byte(c))
	}

	sanitizedUrl := strBuilder.String()
	if suffixSlash && sanitizedUrl[len(sanitizedUrl)-1] != '/' {
		sanitizedUrl += "/"
	}

	if !suffixSlash && sanitizedUrl[len(sanitizedUrl)-1] == '/' {
		sanitizedUrl = sanitizedUrl[:len(sanitizedUrl)-1]
	}

	return sanitizedUrl
}
