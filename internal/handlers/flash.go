package handlers

import (
	"net/http"
	"net/url"
	"time"
)

// flashCookieName est le nom du cookie portant le message de confirmation.
const flashCookieName = "flash"

// setFlash prépare un message affiché après la prochaine redirection.
//
// Le schéma est celui du « POST puis redirection » : après une création ou une
// suppression, on redirige au lieu de rendre une page. L'utilisateur peut
// ainsi actualiser sans rejouer l'action, et le bouton « retour » ne propose
// pas de renvoyer le formulaire.
//
// Le message doit donc survivre à la redirection. Il est porté par un cookie
// de courte durée, lu et immédiatement effacé au rendu suivant. Cela évite de
// maintenir un stockage de sessions côté client pour une seule chaîne de
// caractères.
//
// La valeur est encodée en URL car un cookie ne peut contenir ni espace ni
// caractère accentué.
func setFlash(w http.ResponseWriter, message string) {
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    url.QueryEscape(message),
		Path:     "/",
		MaxAge:   int(time.Minute.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// takeFlash lit le message en attente et demande au navigateur de l'oublier.
//
// La lecture est destructrice : un message de confirmation qui resterait
// affiché après un rechargement de page serait trompeur.
func takeFlash(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie(flashCookieName)
	if err != nil || cookie.Value == "" {
		return ""
	}

	// Effacement immédiat, avant même le rendu.
	http.SetCookie(w, &http.Cookie{
		Name:     flashCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	message, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		// Cookie corrompu ou forgé : on l'ignore plutôt que d'afficher du
		// contenu illisible.
		return ""
	}

	return message
}
