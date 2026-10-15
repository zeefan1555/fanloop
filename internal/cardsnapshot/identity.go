package cardsnapshot

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	prefix      = ".fanloop/card/"
	stampLayout = "20060102T150405.000000-0700"
	stampSample = "20260807T004805.880691+0800"
)

func Path(now time.Time, sourceEventID string, collision int) string {
	stamp := now.In(time.FixedZone("UTC+8", 8*60*60)).Format(stampLayout)
	base := stamp + "-" + SourceDigest(sourceEventID)
	if collision > 0 {
		base = fmt.Sprintf("%s-%d", base, collision)
	}
	return filepath.ToSlash(filepath.Join(".fanloop", "card", base+".json"))
}

func SourceDigest(sourceEventID string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(sourceEventID)))
}

func MatchesSourceEvent(relative, sourceEventID string) bool {
	if !ValidPath(relative) {
		return false
	}
	name := filepath.Base(filepath.FromSlash(relative))
	stem := strings.TrimSuffix(name, ".json")
	rest := stem[len(stampSample):]
	marker := "-" + SourceDigest(sourceEventID)
	return strings.HasPrefix(rest, marker) && validCollision(rest[len(marker):])
}

func ValidPath(relative string) bool {
	if !strings.HasPrefix(relative, prefix) || strings.Contains(strings.TrimPrefix(relative, prefix), "/") || !strings.HasSuffix(relative, ".json") {
		return false
	}
	stem := strings.TrimSuffix(strings.TrimPrefix(relative, prefix), ".json")
	if len(stem) < len(stampSample)+1+64 {
		return false
	}
	if _, err := time.Parse(stampLayout, stem[:len(stampSample)]); err != nil {
		return false
	}
	rest := stem[len(stampSample):]
	if rest[0] != '-' || !lowerHex(rest[1:1+64]) {
		return false
	}
	return validCollision(rest[1+64:])
}

func lowerHex(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return false
		}
	}
	return true
}

func validCollision(value string) bool {
	if value == "" {
		return true
	}
	if !strings.HasPrefix(value, "-") || len(value) == 1 {
		return false
	}
	nonZero := false
	for _, character := range value[1:] {
		if character < '0' || character > '9' {
			return false
		}
		nonZero = nonZero || character != '0'
	}
	return nonZero
}
