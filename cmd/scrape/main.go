// cmd/scrape scrapes pokebase.app for the Champions-specific Pokemon roster.
// Output: seed/pokebase-raw.json  (gitignored)
// Rate-limited to 1 req/sec. Data credited to pokebase.app in the app footer.
//
// Usage:
//
//	go run ./cmd/scrape           -- full scrape
//	go run ./cmd/scrape --debug   -- saves raw HTML to seed/debug/ and exits
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseURL    = "https://pokebase.app"
	listPath   = "/pokemon-champions/pokemon"
	totalPages = 4
	rateLimit  = time.Second
)

// --- Output types ---

type RawMove struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	Category    string `json:"category"` // physical | special | status
	Power       *int   `json:"power"`
	Accuracy    *int   `json:"accuracy"`
	PP          int    `json:"pp"`
}

type RawAbility struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

type RawBaseStats struct {
	HP      int `json:"hp"`
	Attack  int `json:"attack"`
	Defense int `json:"defense"`
	SpAtk   int `json:"sp_atk"`
	SpDef   int `json:"sp_def"`
	Speed   int `json:"speed"`
}

type RawPokemon struct {
	Slug        string       `json:"slug"`
	DexNumber   int          `json:"dex_number"`
	DisplayName string       `json:"display_name"`
	SpriteURL   string       `json:"sprite_url"`
	Types       []string     `json:"types"`
	BaseStats   RawBaseStats `json:"base_stats"`
	Abilities   []RawAbility `json:"abilities"`
	Moves       []RawMove    `json:"moves"`
}

type RawOutput struct {
	ScrapedAt string       `json:"scraped_at"`
	Source    string       `json:"source"`
	Pokemon   []RawPokemon `json:"pokemon"`
}

// statRe matches "108<!-- --> / <!-- -->183" in raw Next.js SSR HTML.
// Stats appear in order: HP, ATK, DEF, Sp. Atk, Sp. Def, SPD.
var statRe = regexp.MustCompile(`(\d+)<!-- --> / <!-- -->(\d+)`)

// dexRe matches "#<!-- -->445" in raw Next.js SSR HTML.
var dexRe = regexp.MustCompile(`#<!-- -->(\d+)`)

func main() {
	debug := flag.Bool("debug", false, "save raw HTML to seed/debug/ for selector inspection, then exit")
	flag.Parse()

	if *debug {
		if err := os.MkdirAll("seed/debug", 0755); err != nil {
			log.Fatalf("mkdir seed/debug: %v", err)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	throttle := time.NewTicker(rateLimit)
	defer throttle.Stop()

	log.Println("collecting Pokemon slugs from list pages...")
	slugs, err := collectSlugs(client, throttle, *debug)
	if err != nil {
		log.Fatalf("collect slugs: %v", err)
	}
	log.Printf("found %d unique Pokemon slugs", len(slugs))

	if *debug {
		// Scrape one detail page for selector inspection
		<-throttle.C
		log.Println("debug: scraping garchomp detail page...")
		body, err := fetchRaw(client, baseURL+listPath+"/garchomp")
		if err != nil {
			log.Printf("WARN: could not fetch garchomp: %v", err)
		} else {
			if err := os.WriteFile("seed/debug/garchomp.html", []byte(body), 0644); err != nil {
				log.Printf("WARN: save debug html: %v", err)
			}
			log.Println("saved seed/debug/garchomp.html")
		}
		log.Println("debug mode: exiting. Check seed/debug/ for HTML samples.")
		return
	}

	var pokemon []RawPokemon
	for i, slug := range slugs {
		<-throttle.C
		log.Printf("[%d/%d] scraping %s", i+1, len(slugs), slug)
		p, err := scrapePokemon(client, slug)
		if err != nil {
			log.Printf("WARN: failed to scrape %s: %v -- skipping", slug, err)
			continue
		}
		pokemon = append(pokemon, *p)
	}

	out := RawOutput{
		ScrapedAt: time.Now().UTC().Format(time.RFC3339),
		Source:    "https://pokebase.app/pokemon-champions/pokemon",
		Pokemon:   pokemon,
	}

	if err := os.MkdirAll("seed", 0755); err != nil {
		log.Fatalf("mkdir seed: %v", err)
	}

	f, err := os.Create("seed/pokebase-raw.json")
	if err != nil {
		log.Fatalf("create output: %v", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		log.Fatalf("encode output: %v", err)
	}

	log.Printf("done. wrote %d Pokemon to seed/pokebase-raw.json", len(pokemon))
}

func collectSlugs(client *http.Client, throttle *time.Ticker, debug bool) ([]string, error) {
	seen := make(map[string]bool)
	var slugs []string

	for page := 1; page <= totalPages; page++ {
		<-throttle.C
		url := fmt.Sprintf("%s%s?page=%d", baseURL, listPath, page)
		log.Printf("fetching list page %d: %s", page, url)

		body, err := fetchRaw(client, url)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", page, err)
		}

		if debug && page == 1 {
			if err := os.WriteFile("seed/debug/list-page-1.html", []byte(body), 0644); err != nil {
				log.Printf("WARN: save debug html: %v", err)
			}
			log.Println("saved seed/debug/list-page-1.html")
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("parse page %d: %w", page, err)
		}

		doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			slug := extractPokemonSlug(href)
			if slug == "" || seen[slug] {
				return
			}
			seen[slug] = true
			slugs = append(slugs, slug)
		})
	}

	return slugs, nil
}

func scrapePokemon(client *http.Client, slug string) (*RawPokemon, error) {
	url := fmt.Sprintf("%s%s/%s", baseURL, listPath, slug)
	body, err := fetchRaw(client, url)
	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	p := &RawPokemon{Slug: slug}

	// Display name from h1 (e.g. "Garchomp")
	p.DisplayName = strings.TrimSpace(doc.Find("h1").First().Text())
	if p.DisplayName == "" {
		p.DisplayName = slugToDisplay(slug)
		log.Printf("  WARN: no h1 name for %s, using slug-derived: %s", slug, p.DisplayName)
	}

	// National dex number from "#<!-- -->445" pattern in raw SSR HTML
	if m := dexRe.FindStringSubmatch(body); m != nil {
		p.DexNumber, _ = strconv.Atoi(m[1])
	}

	// Sprite URL: first img with alt=DisplayName
	doc.Find("img[alt]").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if alt, _ := s.Attr("alt"); alt == p.DisplayName {
			p.SpriteURL, _ = s.Attr("src")
			return false
		}
		return true
	})

	// Types: images inside buttons that have data-state but no aria-label.
	// The Pokemon type icons are in <button data-state="closed"><img alt="Dragon">.
	// Move category buttons have aria-label="Physical"; type filter buttons have type="button"
	// but no data-state attribute. Only data-state+no-aria-label matches the type icons.
	doc.Find("button[data-state]:not([aria-label]) img[alt]").Each(func(_ int, s *goquery.Selection) {
		alt, _ := s.Attr("alt")
		t := strings.ToLower(alt)
		if isValidType(t) && !contains(p.Types, t) {
			p.Types = append(p.Types, t)
		}
	})

	// Base stats from raw HTML (preserves "108<!-- --> / <!-- -->183" comment pattern)
	p.BaseStats = parseStats(body)

	// Abilities from list items
	p.Abilities = parseAbilities(doc)

	// Moves from table rows
	p.Moves = parseMoves(doc)

	if len(p.Types) == 0 {
		log.Printf("  WARN: no types for %s", slug)
	}
	if len(p.Abilities) == 0 {
		log.Printf("  WARN: no abilities for %s", slug)
	}
	if len(p.Moves) == 0 {
		log.Printf("  WARN: no moves for %s", slug)
	}

	return p, nil
}

// parseStats extracts the 6 base stats from raw SSR HTML.
// Next.js renders stat values as "108<!-- --> / <!-- -->183" (raw / lvl-50).
// Stats always appear in order: HP, ATK, DEF, Sp. Atk, Sp. Def, SPD.
func parseStats(body string) RawBaseStats {
	var s RawBaseStats
	order := []*int{&s.HP, &s.Attack, &s.Defense, &s.SpAtk, &s.SpDef, &s.Speed}
	for i, m := range statRe.FindAllStringSubmatch(body, -1) {
		if i >= 6 {
			break
		}
		v, _ := strconv.Atoi(m[1])
		*order[i] = v
	}
	return s
}

// parseAbilities extracts abilities from <li class="py-3"> elements.
// Name comes from [aria-label], description from <p>.
func parseAbilities(doc *goquery.Document) []RawAbility {
	var abilities []RawAbility
	doc.Find(`li[class*="py-3"]`).Each(func(_ int, s *goquery.Selection) {
		nameEl := s.Find("[aria-label]").First()
		name := strings.TrimSpace(nameEl.AttrOr("aria-label", ""))
		if name == "" {
			name = strings.TrimSpace(nameEl.Text())
		}
		if name == "" || len(name) > 60 || strings.Contains(name, ",") || strings.Contains(name, "%") {
			return
		}
		desc := strings.TrimSpace(s.Find("p").First().Text())
		abilities = append(abilities, RawAbility{
			Name:        toSlug(name),
			DisplayName: name,
			Description: desc,
		})
	})
	return abilities
}

// parseMoves extracts moves from div.table-row elements.
// Each row: span.table-cell[0] has type img + name link + category button;
// span.table-cell[1] = power, [2] = accuracy (with %), [3] = PP.
func parseMoves(doc *goquery.Document) []RawMove {
	var moves []RawMove
	seen := make(map[string]bool)

	doc.Find("div.table-row").Each(func(_ int, row *goquery.Selection) {
		link := row.Find(`a[href*="/pokemon-champions/moves/"]`).First()
		displayName := strings.TrimSpace(link.Text())
		if displayName == "" {
			return
		}
		slug := extractMoveSlug(link.AttrOr("href", ""))
		if slug == "" || seen[slug] {
			return
		}

		cells := row.Find("span.table-cell")
		if cells.Length() < 4 {
			return
		}

		nameCell := cells.Eq(0)
		moveType := strings.ToLower(nameCell.Find("img[alt]").First().AttrOr("alt", ""))
		category := strings.ToLower(nameCell.Find("button[aria-label]").First().AttrOr("aria-label", ""))

		if !isValidType(moveType) || !isValidCategory(category) {
			return
		}

		powerText := strings.TrimSpace(cells.Eq(1).Text())
		accText := strings.TrimSuffix(strings.TrimSpace(cells.Eq(2).Text()), "%")
		ppText := strings.TrimSpace(cells.Eq(3).Text())

		seen[slug] = true
		moves = append(moves, RawMove{
			Name:        slug,
			DisplayName: displayName,
			Type:        moveType,
			Category:    category,
			Power:       parseIntPtr(powerText),
			Accuracy:    parseIntPtr(accText),
			PP:          parseInt(ppText),
		})
	})

	return moves
}

// --- Helpers ---

// fetchRaw fetches a URL and returns the raw response body as a string.
// Preserving the raw HTML is important for stat parsing (Next.js SSR comments).
func fetchRaw(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}
	return string(b), nil
}

func extractPokemonSlug(href string) string {
	const prefix = "/pokemon-champions/pokemon/"
	if !strings.HasPrefix(href, prefix) {
		return ""
	}
	slug := strings.TrimPrefix(href, prefix)
	if slug == "" || strings.Contains(slug, "?") || strings.Contains(slug, "/") {
		return ""
	}
	return slug
}

func extractMoveSlug(href string) string {
	const prefix = "/pokemon-champions/moves/"
	if !strings.HasPrefix(href, prefix) {
		return ""
	}
	slug := strings.TrimPrefix(href, prefix)
	if slug == "" || strings.Contains(slug, "/") {
		return ""
	}
	return slug
}

func toSlug(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
}

func slugToDisplay(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func isValidType(t string) bool {
	switch t {
	case "normal", "fire", "water", "electric", "grass", "ice",
		"fighting", "poison", "ground", "flying", "psychic", "bug",
		"rock", "ghost", "dragon", "dark", "steel", "fairy":
		return true
	}
	return false
}

func isValidCategory(c string) bool {
	return c == "physical" || c == "special" || c == "status"
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

func parseIntPtr(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "—" || s == "N/A" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}
