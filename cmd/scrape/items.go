package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const itemsPath = "/pokemon-champions/items"

// RawItem is one entry from pokebase.app's items catalog.
type RawItem struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	SpriteURL   string `json:"sprite_url"`
	Category    string `json:"category"` // berry | held-item | mega-evolution
}

type RawItemsOutput struct {
	ScrapedAt string    `json:"scraped_at"`
	Source    string    `json:"source"`
	Items     []RawItem `json:"items"`
}

// itemJSON mirrors the shape embedded in pokebase.app's Next.js RSC payload.
type itemJSON struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Icon        struct {
		URL string `json:"url"`
	} `json:"icon"`
	Category string `json:"category"`
}

// scrapeItems fetches the items catalog page. Unlike the Pokemon list, the
// full catalog (148 items) ships as one page -- no pagination, no per-item
// detail fetch needed. The data is embedded as an escaped JSON string inside
// the page's Next.js RSC payload rather than rendered HTML, so we extract and
// unescape that fragment instead of using goquery selectors.
func scrapeItems(client *http.Client) (*RawItemsOutput, error) {
	body, err := fetchRaw(client, baseURL+itemsPath)
	if err != nil {
		return nil, fmt.Errorf("fetch items page: %w", err)
	}

	chunk, err := extractDocsArray(body, `\"category\":\"held-item\"`)
	if err != nil {
		return nil, fmt.Errorf("locate items docs array: %w", err)
	}

	unescaped := unescapeJSString(chunk)

	var items []itemJSON
	if err := json.Unmarshal([]byte(unescaped), &items); err != nil {
		return nil, fmt.Errorf("parse items JSON: %w", err)
	}

	out := &RawItemsOutput{
		ScrapedAt: time.Now().UTC().Format(time.RFC3339),
		Source:    baseURL + itemsPath,
	}
	for _, it := range items {
		out.Items = append(out.Items, RawItem{
			Name:        it.Slug,
			DisplayName: it.Name,
			Slug:        it.Slug,
			Description: it.Description,
			SpriteURL:   it.Icon.URL,
			Category:    it.Category,
		})
	}
	return out, nil
}

// extractDocsArray finds the `"docs":[...]` array that contains marker, and
// returns the raw (still JS-escaped) array substring including the brackets.
func extractDocsArray(body, marker string) (string, error) {
	before, _, found := strings.Cut(body, marker)
	if !found {
		return "", fmt.Errorf("marker %q not found in page", marker)
	}
	markerIdx := len(before)

	const docsKey = `\"docs\":[`
	docsIdx := strings.LastIndex(body[:markerIdx], docsKey)
	if docsIdx == -1 {
		return "", fmt.Errorf("no preceding docs array found")
	}
	arrStart := docsIdx + len(docsKey) - 1 // index of the '['

	depth := 0
	for i := arrStart; i < len(body); i++ {
		switch body[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return body[arrStart : i+1], nil
			}
		}
	}
	return "", fmt.Errorf("unbalanced brackets in docs array")
}

// unescapeJSString reverses the single level of backslash-escaping used to
// embed a JSON payload inside a JS string literal (e.g. `\"` -> `"`).
func unescapeJSString(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case '"':
				b.WriteByte('"')
				i++
				continue
			case '\\':
				b.WriteByte('\\')
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func writeItemsJSON(path string, out *RawItemsOutput) error {
	if err := os.MkdirAll("seed", 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
