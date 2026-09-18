package handlers

import (
	"net/http"
	"net/url"
	"time"
)

// flashCookieName est le nom du cookie portant le message de confirmation
const flashCookieName = "flash"

// flashErrorCookieName est le nom du cookie portant un message d'erreur à afficher après redirection
const flashErrorCookieName = "flash_error"

// setFlash prépare un message de confirmation affiché après la prochaine redirection
func setFlash(w http.ResponseWriter, message string) {
	writeFlash(w, flashCookieName, message)
}

// setFlashError prépare un message d'erreur affiché après la prochaine redirection
func setFlashError(w http.ResponseWriter, message string) {
	writeFlash(w, flashErrorCookieName, message)
}

// takeFlash lit le message de confirmation en attente et demande au navigateur de l'oublier
func takeFlash(w http.ResponseWriter, r *http.Request) string {
	return readFlash(w, r, flashCookieName)
}

// takeFlashError lit le message d'erreur en attente et demande au navigateur de l'oublier
func takeFlashError(w http.ResponseWriter, r *http.Request) string {
	return readFlash(w, r, flashErrorCookieName)
}

// writeFlash dépose un message dans un cookie de courte durée
func writeFlash(w http.ResponseWriter, name, message string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    url.QueryEscape(message),
		Path:     "/",
		MaxAge:   int(time.Minute.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// readFlash lit un message puis l'efface, un message qui survivrait à un rechargement serait trompeur
func readFlash(w http.ResponseWriter, r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil || cookie.Value == "" {
		return ""
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	message, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return ""
	}

	return message
}
