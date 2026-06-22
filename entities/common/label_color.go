package common

import (
	"strings"

	"github.com/git-bug/git-bug/repository"
)

const labelColorConfigPrefix = "git-bug.label."

func GetLabelColor(cfg repository.ConfigRead, label Label) LabelColor {
	key := labelColorConfigPrefix + strings.ToLower(label.String()) + ".color"
	colorStr, err := cfg.ReadString(key)
	if err != nil {
		return label.DefaultColor()
	}

	color, err := ParseHexColor(colorStr)
	if err != nil {
		return DefaultLabelColor
	}

	return color
}

func SetLabelColor(cfg repository.ConfigWrite, label Label, hexColor string) error {
	_, err := ParseHexColor(hexColor)
	if err != nil {
		return err
	}

	key := labelColorConfigPrefix + strings.ToLower(label.String()) + ".color"
	return cfg.StoreString(key, hexColor)
}

func RemoveLabelColor(cfg repository.ConfigWrite, label Label) error {
	key := labelColorConfigPrefix + strings.ToLower(label.String())
	return cfg.RemoveAll(key)
}

func ListLabelColors(cfg repository.ConfigRead) (map[Label]LabelColor, error) {
	result := make(map[Label]LabelColor)

	entries, err := cfg.ReadAll(labelColorConfigPrefix)
	if err != nil {
		return nil, err
	}

	for key, value := range entries {
		if !strings.HasSuffix(key, ".color") {
			continue
		}

		labelName := strings.TrimPrefix(key, labelColorConfigPrefix)
		labelName = strings.TrimSuffix(labelName, ".color")

		color, err := ParseHexColor(value)
		if err != nil {
			continue
		}

		result[Label(labelName)] = color
	}

	return result, nil
}
