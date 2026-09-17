package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// findCookie retrouve un cookie parmi ceux déposés dans la réponse.
func findCookie(t *testing.T, recorder *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}

	t.Fatalf("aucun cookie nommé %q dans la réponse", name)
	return nil
}

// Les attributs du cookie de session sont le point sensible de ce package :
// ce sont eux qui empêchent l'exfiltration du jeton par XSS et son rejeu
// depuis un autre site.
func TestSaveSetsProtectiveCookieAttributes(t *testing.T) {
	recorder := httptest.NewRecorder()
	expiresAt := time.Now().Add(time.Hour)

	Save(recorder, "mon-jeton", expiresAt)

	cookie := findCookie(t, recorder, cookieName)

	if cookie.Value != "mon-jeton" {
		t.Errorf("valeur = %q, attendu %q", cookie.Value, "mon-jeton")
	}

	// Sans HttpOnly, une faille XSS dans une page permettrait de lire le
	// jeton depuis JavaScript et de l'exfiltrer.
	if !cookie.HttpOnly {
		t.Error("HttpOnly = false : le jeton serait lisible depuis JavaScript")
	}

	// Sans SameSite, un site tiers pourrait déclencher une suppression en
	// faisant envoyer le cookie par le navigateur de la victime.
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, attendu SameSiteLaxMode", cookie.SameSite)
	}

	if cookie.Path != "/" {
		t.Errorf("Path = %q, attendu \"/\"", cookie.Path)
	}

	// L'expiration du cookie doit suivre celle du jeton : le navigateur
	// cesse de l'envoyer au moment même où il devient inutilisable.
	if !cookie.Expires.Equal(expiresAt.UTC().Truncate(time.Second)) &&
		cookie.Expires.Sub(expiresAt).Abs() > time.Second {
		t.Errorf("Expires = %v, attendu environ %v", cookie.Expires, expiresAt)
	}
}

func TestTokenReadsCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/spaces", nil)
	request.AddCookie(&http.Cookie{Name: cookieName, Value: "mon-jeton"})

	token, ok := Token(request)
	if !ok {
		t.Fatal("Token() = false alors que le cookie est présent")
	}
	if token != "mon-jeton" {
		t.Errorf("jeton = %q, attendu %q", token, "mon-jeton")
	}
}

func TestTokenReportsMissingSession(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/spaces", nil)

	if _, ok := Token(request); ok {
		t.Error("Token() = true alors qu'aucun cookie n'est présent")
	}
}

// Un cookie présent mais vide ne constitue pas une session : le traiter comme
// valide enverrait un en-tête « Bearer » vide à l'API.
func TestTokenRejectsEmptyCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/spaces", nil)
	request.AddCookie(&http.Cookie{Name: cookieName, Value: ""})

	if _, ok := Token(request); ok {
		t.Error("Token() = true pour un cookie vide")
	}
}

// Effacer un cookie consiste à le réécrire vide et expiré : il n'existe pas
// d'autre moyen pour un serveur de demander à un navigateur de l'oublier.
func TestClearExpiresCookie(t *testing.T) {
	recorder := httptest.NewRecorder()

	Clear(recorder)

	cookie := findCookie(t, recorder, cookieName)

	if cookie.Value != "" {
		t.Errorf("valeur = %q, attendu une chaîne vide", cookie.Value)
	}
	if cookie.MaxAge >= 0 {
		t.Errorf("MaxAge = %d, une valeur négative est attendue pour supprimer le cookie", cookie.MaxAge)
	}
	if !cookie.Expires.Before(time.Now()) {
		t.Errorf("Expires = %v, une date passée est attendue", cookie.Expires)
	}
}

// Vérifie le cycle complet : un jeton déposé puis relu doit être identique.
func TestSaveThenTokenRoundTrip(t *testing.T) {
	recorder := httptest.NewRecorder()
	Save(recorder, "jeton-aller-retour", time.Now().Add(time.Hour))

	request := httptest.NewRequest(http.MethodGet, "/spaces", nil)
	for _, cookie := range recorder.Result().Cookies() {
		request.AddCookie(cookie)
	}

	token, ok := Token(request)
	if !ok {
		t.Fatal("Token() = false après Save()")
	}
	if token != "jeton-aller-retour" {
		t.Errorf("jeton = %q, attendu %q", token, "jeton-aller-retour")
	}
}
