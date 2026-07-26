package services

import (
	"regexp"
	"strconv"
	"strings"
)

// ParsedSearchQuery is the free-text term plus optional year filters from the query string.
// Examples: "skrillex :after:2020", "drake :before:2015", "dualipa :after:2018 :before:2022"
type ParsedSearchQuery struct {
	Text     string
	YearFrom int // inclusive; 0 = unset
	YearTo   int // inclusive; 0 = unset
}

var searchFilterToken = regexp.MustCompile(`(?i)(?:^|\s):(after|from|before|until|to|yearfrom|yearto):(\d{4})(?:\s|$)`)

// ParseSearchQuery strips filter tokens and returns the clean search text + year bounds.
func ParseSearchQuery(raw string) ParsedSearchQuery {
	raw = strings.TrimSpace(raw)
	out := ParsedSearchQuery{}
	if raw == "" {
		return out
	}

	rest := raw
	for {
		loc := searchFilterToken.FindStringSubmatchIndex(rest)
		if loc == nil {
			break
		}
		key := strings.ToLower(rest[loc[2]:loc[3]])
		year, err := strconv.Atoi(rest[loc[4]:loc[5]])
		if err == nil && year >= 1900 && year <= 2100 {
			switch key {
			case "after", "from", "yearfrom":
				out.YearFrom = year
			case "before", "until", "to", "yearto":
				out.YearTo = year
			}
		}
		// Drop the matched token (keep surrounding spaces collapsed later).
		rest = rest[:loc[0]] + " " + rest[loc[1]:]
	}

	out.Text = strings.Join(strings.Fields(rest), " ")
	if out.YearFrom > 0 && out.YearTo > 0 && out.YearFrom > out.YearTo {
		out.YearFrom, out.YearTo = out.YearTo, out.YearFrom
	}
	return out
}

func (p ParsedSearchQuery) HasYearFilter() bool {
	return p.YearFrom > 0 || p.YearTo > 0
}
