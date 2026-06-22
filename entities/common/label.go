package common

import (
	"crypto/sha256"
	"fmt"
	"image/color"
	"regexp"
	"strconv"
	"strings"

	fcolor "github.com/fatih/color"

	"github.com/git-bug/git-bug/util/text"
)

type Label string

func (l Label) String() string {
	return string(l)
}

var DefaultLabelColor = LabelColor{R: 158, G: 158, B: 158, A: 255}

var hexColorRegex = regexp.MustCompile(`^#?([0-9a-fA-F]{8}|[0-9a-fA-F]{6}|[0-9a-fA-F]{4}|[0-9a-fA-F]{3})$`)

func ParseHexColor(s string) (LabelColor, error) {
	s = strings.TrimSpace(s)
	if !hexColorRegex.MatchString(s) {
		return DefaultLabelColor, fmt.Errorf("invalid hex color: %s", s)
	}

	hex := s
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	switch len(hex) {
	case 3:
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2], 'f', 'f'})
	case 4:
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2], hex[3], hex[3]})
	case 6:
		hex = hex + "ff"
	}

	r, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return DefaultLabelColor, fmt.Errorf("invalid hex color: %s", s)
	}
	g, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return DefaultLabelColor, fmt.Errorf("invalid hex color: %s", s)
	}
	b, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return DefaultLabelColor, fmt.Errorf("invalid hex color: %s", s)
	}
	a, err := strconv.ParseUint(hex[6:8], 16, 8)
	if err != nil {
		return DefaultLabelColor, fmt.Errorf("invalid hex color: %s", s)
	}

	return LabelColor{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}, nil
}

func (l Label) DefaultColor() LabelColor {
	colors := []LabelColor{
		{R: 244, G: 67, B: 54, A: 255},   // red
		{R: 233, G: 30, B: 99, A: 255},   // pink
		{R: 156, G: 39, B: 176, A: 255},  // purple
		{R: 103, G: 58, B: 183, A: 255},  // deepPurple
		{R: 63, G: 81, B: 181, A: 255},   // indigo
		{R: 33, G: 150, B: 243, A: 255},  // blue
		{R: 3, G: 169, B: 244, A: 255},   // lightBlue
		{R: 0, G: 188, B: 212, A: 255},   // cyan
		{R: 0, G: 150, B: 136, A: 255},   // teal
		{R: 76, G: 175, B: 80, A: 255},   // green
		{R: 139, G: 195, B: 74, A: 255},  // lightGreen
		{R: 205, G: 220, B: 57, A: 255},  // lime
		{R: 255, G: 235, B: 59, A: 255},  // yellow
		{R: 255, G: 193, B: 7, A: 255},   // amber
		{R: 255, G: 152, B: 0, A: 255},   // orange
		{R: 255, G: 87, B: 34, A: 255},   // deepOrange
		{R: 121, G: 85, B: 72, A: 255},   // brown
		{R: 158, G: 158, B: 158, A: 255}, // grey
		{R: 96, G: 125, B: 139, A: 255},  // blueGrey
	}

	id := 0
	hash := sha256.Sum256([]byte(l))
	for _, char := range hash {
		id = (id + int(char)) % len(colors)
	}

	return colors[id]
}

func (l Label) Color() LabelColor {
	return l.DefaultColor()
}

func (l Label) Validate() error {
	str := string(l)

	if text.Empty(str) {
		return fmt.Errorf("empty")
	}

	if !text.SafeOneLine(str) {
		return fmt.Errorf("label has unsafe characters")
	}

	return nil
}

type LabelColor color.RGBA

func (lc LabelColor) RGBA() color.RGBA {
	return color.RGBA(lc)
}

func (lc LabelColor) Term256() Term256 {
	red := Term256((int(lc.R)*5 + 127) / 255)
	green := Term256((int(lc.G)*5 + 127) / 255)
	blue := Term256((int(lc.B)*5 + 127) / 255)

	return red*36 + green*6 + blue + 16
}

type Term256 int

func (t Term256) Escape() string {
	if fcolor.NoColor {
		return ""
	}
	return fmt.Sprintf("\x1b[38;5;%dm", t)
}

func (t Term256) Unescape() string {
	if fcolor.NoColor {
		return ""
	}
	return "\x1b[0m"
}
