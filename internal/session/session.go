// Package session gère la session de l'utilisateur côté client web.
//
// Le client est un serveur HTTP classique rendu par templates : il n'a pas de
// JavaScript pour porter un jeton en mémoire. Le jeton JWT délivré par l'API
// est donc stocké dans un cookie, et rejoué par le client vers l'API à chaque
// requête.
package session

import (
	"net/http"
	"time"
)

// cookieName est le nom du cookie portant le jeton JWT.
const cookieName = "session_token"

// Save dépose le jeton dans un cookie.
//
// Les attributs du cookie sont les points importants de ce fichier :
//
//   - HttpOnly : le cookie est invisible depuis JavaScript. Même en cas de
//     faille XSS dans une page, le jeton ne peut pas être exfiltré.
//   - SameSite=Lax : le cookie n'est pas envoyé lors de requêtes
//     inter-sites, ce qui bloque les attaques CSRF sur les formulaires de
//     modification et de suppression.
//   - Path=/ : le cookie accompagne toutes les pages du client.
//   - Expires aligné sur l'expiration du jeton : le navigateur oublie le
//     cookie au moment même où le jeton devient invalide, ce qui évite
//     d'envoyer à l'API des requêtes vouées à échouer.
//
// Secure n'est pas activé car le développement se fait en HTTP. En production
// derrière HTTPS, il faudrait le passer à true : c'est signalé dans les
// limites de la documentation technique.
func Save(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// Token retourne le jeton porté par la requête.
// Le second retour est false si l'utilisateur n'a pas de session.
func Token(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

// Clear supprime le cookie de session.
//
// La suppression consiste à réécrire le cookie avec une date d'expiration
// passée et une valeur vide : il n'existe pas d'autre moyen pour un serveur
// de demander à un navigateur d'oublier un cookie.
//
// Le jeton JWT lui-même reste techniquement valide jusqu'à son expiration,
// puisqu'une API sans état ne peut pas révoquer un jeton déjà émis. C'est une
// limite assumée, documentée dans la documentation technique.
func Clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
