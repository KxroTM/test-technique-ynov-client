package render

import (
	"strings"
	"testing"
	"time"
)

func TestTruncateLeavesShortTextUntouched(t *testing.T) {
	const text = "Un titre court"

	if got := truncate(text, 50); got != text {
		t.Errorf("truncate(%q, 50) = %q, attendu %q", text, got, text)
	}
}

func TestTruncateCutsOnWordBoundary(t *testing.T) {
	const text = "Réviser les jointures et les index avant le partiel"

	got := truncate(text, 20)

	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncate = %q, une ellipse est attendue en fin de chaîne", got)
	}

	// La coupe ne doit pas laisser de mot tronqué : le texte retenu, privé
	// de son ellipse, doit se terminer par un mot complet.
	withoutEllipsis := strings.TrimSuffix(got, "…")
	if strings.HasSuffix(withoutEllipsis, " ") {
		t.Errorf("truncate = %q, un espace subsiste avant l'ellipse", got)
	}

	if !strings.HasPrefix(text, withoutEllipsis) {
		t.Errorf("truncate = %q, le début du texte d'origine n'est pas conservé", got)
	}
}

// La découpe doit se faire sur les runes et non sur les octets. Couper une
// chaîne UTF-8 au milieu d'un caractère accentué produirait un octet invalide,
// affiché comme un losange noir par le navigateur.
func TestTruncateCutsOnRunesNotBytes(t *testing.T) {
	// Chaque caractère accentué occupe deux octets en UTF-8 : cette chaîne
	// fait 10 runes mais 20 octets.
	const text = "éééééééééé"

	if len(text) == len([]rune(text)) {
		t.Fatal("la chaîne de test doit contenir des caractères multi-octets")
	}

	got := truncate(text, 5)

	// Aucun caractère de remplacement ne doit apparaître : sa présence
	// signalerait une séquence UTF-8 invalide.
	if strings.ContainsRune(got, '�') {
		t.Errorf("truncate = %q, la chaîne a été coupée au milieu d'un caractère", got)
	}

	// 5 runes conservées, plus l'ellipse.
	if want := "ééééé…"; got != want {
		t.Errorf("truncate = %q, attendu %q", got, want)
	}
}

func TestTruncateHandlesTextWithoutSpaces(t *testing.T) {
	const text = "abcdefghijklmnop"

	got := truncate(text, 5)

	// Sans espace où reculer, la coupe se fait net à la longueur demandée.
	if want := "abcde…"; got != want {
		t.Errorf("truncate = %q, attendu %q", got, want)
	}
}

func TestFormatDate(t *testing.T) {
	// La date est construite dans le fuseau local car formatDate convertit
	// en heure locale avant de formater.
	moment := time.Date(2026, time.March, 9, 14, 5, 0, 0, time.Local)

	if want, got := "09/03/2026 à 14:05", formatDate(moment); got != want {
		t.Errorf("formatDate = %q, attendu %q", got, want)
	}
}
