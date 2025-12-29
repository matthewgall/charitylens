package templates

import (
	"embed"
	"html/template"
	"io/fs"
	"strconv"
	"strings"
)

//go:embed *.html
var templateFS embed.FS

// Templates holds all parsed templates
var Templates *template.Template

// formatCurrency formats a number with comma separators
func formatCurrency(n float64) string {
	// Convert to string with 0 decimal places
	str := strconv.FormatFloat(n, 'f', 0, 64)

	// Handle negative numbers
	negative := false
	if strings.HasPrefix(str, "-") {
		negative = true
		str = str[1:]
	}

	// Add commas
	var result strings.Builder
	length := len(str)
	for i, digit := range str {
		if i > 0 && (length-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(digit)
	}

	if negative {
		return "-" + result.String()
	}
	return result.String()
}

// ensureAbsoluteURL ensures a URL is absolute with https:// prefix
// If the URL doesn't have a scheme, https:// is prepended
func ensureAbsoluteURL(url string) string {
	if url == "" {
		return ""
	}

	// Trim whitespace
	url = strings.TrimSpace(url)

	// Check if it already has a scheme
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}

	// Remove leading slashes or www. if present
	url = strings.TrimPrefix(url, "//")

	// Add https:// prefix
	return "https://" + url
}

// titleCase converts a name to proper title case
// Handles all caps names and preserves certain uppercase elements like initials
func titleCase(s string) string {
	if s == "" {
		return s
	}

	// Words that should remain lowercase (unless at the start)
	minorWords := map[string]bool{
		"of": true, "and": true, "the": true, "de": true, "van": true, "von": true,
		"da": true, "di": true, "del": true, "della": true,
	}

	// Common post-nominal letters and honors that should stay uppercase
	postNominals := map[string]bool{
		// Academic degrees
		"MA": true, "BA": true, "BSC": true, "MSC": true, "MBA": true, "PHD": true,
		"MD": true, "LLB": true, "LLM": true, "BED": true, "MED": true,
		// Professional qualifications
		"FCA": true, "ACA": true, "ACCA": true, "FCCA": true, "CPA": true,
		"CIPFA": true, "CIMA": true, "FCMA": true, "FRICS": true, "MRICS": true,
		// Honors
		"OBE": true, "MBE": true, "CBE": true, "KBE": true, "DBE": true,
		"QC": true, "KC": true, "DL": true, "JP": true,
		// Medical
		"FRCP": true, "MRCP": true, "FRCS": true, "MRCS": true, "FRCPCH": true,
		// Academic/Scientific
		"FRS": true,
		// Engineering
		"CEng": true, "FREng": true, "IEng": true,
		// Other common
		"RN": true, "MP": true, "MSP": true, "AM": true,
	}

	// Split into words
	words := strings.Fields(s)
	result := make([]string, len(words))

	for i, word := range words {
		// Check if it's a known post-nominal (all caps version)
		upperWord := strings.ToUpper(word)
		if postNominals[upperWord] {
			result[i] = upperWord
			continue
		}

		// Check for hyphenated names
		if strings.Contains(word, "-") {
			parts := strings.Split(word, "-")
			for j, part := range parts {
				if part == "" {
					continue
				}
				parts[j] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
			}
			result[i] = strings.Join(parts, "-")
			continue
		}

		// Check for names with apostrophes (O'Brien, D'Angelo)
		if strings.Contains(word, "'") {
			parts := strings.Split(word, "'")
			for j, part := range parts {
				if part == "" {
					continue
				}
				parts[j] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
			}
			result[i] = strings.Join(parts, "'")
			continue
		}

		// Check if it's a minor word (and not the first word)
		lowerWord := strings.ToLower(word)
		if i > 0 && minorWords[lowerWord] {
			result[i] = lowerWord
			continue
		}

		// Standard title case
		result[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
	}

	return strings.Join(result, " ")
}

func init() {
	var err error

	// Create function map with custom functions
	funcMap := template.FuncMap{
		"formatCurrency":    formatCurrency,
		"titleCase":         titleCase,
		"ensureAbsoluteURL": ensureAbsoluteURL,
	}

	// Parse templates with custom functions
	Templates, err = template.New("").Funcs(funcMap).ParseFS(templateFS, "*.html")
	if err != nil {
		panic("Failed to parse templates: " + err.Error())
	}
}

// FS returns the embedded filesystem for serving static files
func FS() fs.FS {
	return templateFS
}
