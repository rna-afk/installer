package main

import "strings"

// The application title is drawn with a deliberately tiny block font: three
// rows are enough to read as a wordmark without crowding the header on short
// terminals.
const (
	bannerHeight = 3
	bannerGap    = 1
)

// appBannerText is the wordmark rendered at the top of every screen.
const appBannerText = "OPENSHIFT INSTALLER"

// bannerFont maps the uppercase letters used by appBannerText to their
// three-row block glyphs. Every row of a glyph must have the same width; most
// letters are three cells wide, but N is four so its diagonal reads as a
// diagonal instead of an H. The blank entry gives word separators a letter's
// width.
var bannerFont = map[rune][bannerHeight]string{
	' ': {"   ", "   ", "   "},
	'A': {"█▀█", "█▀█", "▀ ▀"},
	'E': {"█▀▀", "█▀▀", "▀▀▀"},
	'F': {"█▀▀", "█▀▀", "▀  "},
	'H': {"█ █", "█▀█", "▀ ▀"},
	'I': {" █ ", " █ ", " ▀ "},
	'L': {"█  ", "█  ", "▀▀▀"},
	'N': {"█▄ █", "█ ▀█", "▀  ▀"},
	'O': {"█▀█", "█ █", "▀▀▀"},
	'P': {"█▀█", "█▀▀", "▀  "},
	'R': {"█▀█", "█▀▄", "▀ ▀"},
	'S': {"█▀▀", "▀▀█", "▀▀▀"},
	'T': {"▀█▀", " █ ", " ▀ "},
}

// renderBanner draws text in the block font. It reports false when the text
// contains a character the font does not cover, so callers can fall back to
// plain text.
func renderBanner(text string) (string, bool) {
	rows := make([]strings.Builder, bannerHeight)
	for i, r := range text {
		glyph, ok := bannerFont[r]
		if !ok {
			return "", false
		}
		width := len([]rune(glyph[0]))
		for row := 0; row < bannerHeight; row++ {
			if len([]rune(glyph[row])) != width {
				return "", false
			}
		}
		if i > 0 {
			for row := range rows {
				rows[row].WriteString(strings.Repeat(" ", bannerGap))
			}
		}
		for row := range rows {
			rows[row].WriteString(glyph[row])
		}
	}

	lines := make([]string, bannerHeight)
	for row := range rows {
		lines[row] = rows[row].String()
	}
	return strings.Join(lines, "\n"), true
}

// bannerWidth is the number of cells renderBanner needs for text.
func bannerWidth(text string) int {
	width := 0
	for i, r := range text {
		glyph, ok := bannerFont[r]
		if !ok {
			return 0
		}
		if i > 0 {
			width += bannerGap
		}
		width += len([]rune(glyph[0]))
	}
	return width
}
