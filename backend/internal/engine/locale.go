package engine

import "strings"

// Locale keeps stored language codes interoperable with BCP 47. KZ is a UI label.
func Locale(locale string) string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "kk", "kz":
		return "kk"
	case "en", "eng":
		return "en"
	default:
		return "ru"
	}
}

func Text(locale, ru, kk, en string) string {
	switch Locale(locale) {
	case "kk":
		return kk
	case "en":
		return en
	default:
		return ru
	}
}
