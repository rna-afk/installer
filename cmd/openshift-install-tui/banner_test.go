package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestBannerFontIsConsistent(t *testing.T) {
	for r, glyph := range bannerFont {
		width := len([]rune(glyph[0]))
		if width == 0 {
			t.Fatalf("glyph %q is empty", r)
		}
		for row, line := range glyph {
			if got := len([]rune(line)); got != width {
				t.Fatalf("glyph %q row %d is %d cells wide, want %d: %q", r, row, got, width, line)
			}
		}
	}
}

func TestBannerGlyphsAreDistinct(t *testing.T) {
	seen := map[[bannerHeight]string]rune{}
	for r, glyph := range bannerFont {
		if r == ' ' {
			continue
		}
		if other, ok := seen[glyph]; ok {
			t.Fatalf("glyphs %q and %q are identical", other, r)
		}
		seen[glyph] = r
	}
}

func TestBannerFitsEightyColumns(t *testing.T) {
	if got := bannerWidth(appBannerText); got > 78 {
		t.Fatalf("banner width = %d, want at most 78 so it fits an 80-column terminal", got)
	}
}

func TestRenderBanner(t *testing.T) {
	banner, ok := renderBanner(appBannerText)
	if !ok {
		t.Fatalf("banner text %q is not covered by the font", appBannerText)
	}

	lines := strings.Split(banner, "\n")
	if len(lines) != bannerHeight {
		t.Fatalf("banner has %d lines, want %d", len(lines), bannerHeight)
	}
	for _, line := range lines {
		if got, want := lipgloss.Width(line), bannerWidth(appBannerText); got != want {
			t.Fatalf("banner line width = %d, want %d: %q", got, want, line)
		}
	}
}

func TestRenderBannerRejectsUnknownGlyphs(t *testing.T) {
	if _, ok := renderBanner("OPENSHIFT?"); ok {
		t.Fatal("expected unknown glyph to be rejected")
	}
}
