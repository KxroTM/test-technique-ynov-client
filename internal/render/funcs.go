package render

import (
	"fmt"
	"html/template"
	"strings"
	"time"
	"unicode"
)

// templateFuncs retourne les fonctions utilisables depuis les gabarits
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatDate":  formatDate,
		"relativeAge": relativeAge,
		"truncate":    truncate,
		"initial":     initial,
		"plural":      plural,
	}
}

// formatDate met une date au format jour/mois/année suivi de l'heure
func formatDate(value time.Time) string {
	return value.Local().Format("02/01/2006 à 15:04")
}

// relativeAge exprime l'ancienneté d'une date en langage courant
func relativeAge(value time.Time) string {
	elapsed := time.Since(value)

	switch {
	case elapsed < time.Minute:
		return "à l'instant"
	case elapsed < time.Hour:
		return fmt.Sprintf("il y a %d min", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		return fmt.Sprintf("il y a %d h", int(elapsed.Hours()))
	case elapsed < 48*time.Hour:
		return "hier"
	case elapsed < 7*24*time.Hour:
		return fmt.Sprintf("il y a %d j", int(elapsed.Hours()/24))
	default:
		return value.Local().Format("02/01/2006")
	}
}

// truncate raccourcit un texte à la longueur demandée
func truncate(text string, maxLength int) string {
	runes := []rune(text)
	if len(runes) <= maxLength {
		return text
	}

	cut := runes[:maxLength]

	for i := len(cut) - 1; i >= 0; i-- {
		if cut[i] == ' ' {
			cut = cut[:i]
			break
		}
	}

	return string(cut) + "…"
}

// initial retourne la première lettre d'un nom
func initial(name string) string {
	for _, r := range strings.TrimSpace(name) {
		return string(unicode.ToUpper(r))
	}
	return "?"
}

// plural accorde un nom selon un nombre et le préfixe de ce nombre
func plural(count int, singular, pluralForm string) string {
	if count <= 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, pluralForm)
}
