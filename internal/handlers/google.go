package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/KxroTM/test-technique-ynov-client/internal/session"
)

// googleAuthEndpoint est la page de consentement vers laquelle l'utilisateur est redirigé
const googleAuthEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"

// stateCookieName porte le jeton anti-rejeu le temps de l'aller-retour vers Google
const stateCookieName = "google_state"

// StartGoogleLogin redirige l'utilisateur vers la page de consentement Google
func (h *Handler) StartGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !h.googleEnabled {
		h.notFound(w, r, nil)
		return
	}

	state, err := randomState()
	if err != nil {
		log.Printf("génération du state Google : %v", err)
		h.serverError(w, r, nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/auth/google",
		MaxAge:   int((10 * time.Minute).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	query := url.Values{
		"client_id":     {h.googleClientID},
		"redirect_uri":  {h.googleRedirectURI(r)},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
		"prompt":        {"select_account"},
	}

	http.Redirect(w, r, googleAuthEndpoint+"?"+query.Encode(), http.StatusSeeOther)
}

// CompleteGoogleLogin traite le retour de Google et ouvre la session
func (h *Handler) CompleteGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if !h.googleEnabled {
		h.notFound(w, r, nil)
		return
	}

	clearStateCookie(w)

	if reason := r.URL.Query().Get("error"); reason != "" {
		h.failGoogleLogin(w, r, "Connexion Google annulée.", nil)
		return
	}

	if !stateMatches(r) {
		h.failGoogleLogin(w, r, "La connexion Google a expiré ou n'a pas pu être vérifiée. Réessayez.", nil)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		h.failGoogleLogin(w, r, "Google n'a pas transmis de code d'autorisation.", nil)
		return
	}

	result, err := h.api.LoginWithGoogle(r.Context(), code, h.googleRedirectURI(r))
	if err != nil {
		h.failGoogleLogin(w, r, "La connexion Google a échoué.", err)
		return
	}

	session.Save(w, result.Token, result.ExpiresAt)
	http.Redirect(w, r, "/spaces", http.StatusSeeOther)
}

// failGoogleLogin ramène l'utilisateur au formulaire de connexion avec une explication
func (h *Handler) failGoogleLogin(w http.ResponseWriter, r *http.Request, message string, err error) {
	if err != nil {
		log.Printf("connexion Google : %v", err)
	}

	setFlashError(w, message)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// googleRedirectURI reconstruit l'adresse de retour
func (h *Handler) googleRedirectURI(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/auth/google/callback"
}

// randomState produit une valeur imprévisible liant la demande de connexion à son retour
func randomState() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// stateMatches compare le state renvoyé par Google à celui déposé avant la redirection
func stateMatches(r *http.Request) bool {
	cookie, err := r.Cookie(stateCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	returned := r.URL.Query().Get("state")
	if returned == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(returned)) == 1
}

// clearStateCookie retire le cookie anti-rejeu
func clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/auth/google",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
