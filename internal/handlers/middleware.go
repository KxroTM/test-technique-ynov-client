package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
	"github.com/KxroTM/test-technique-ynov-client/internal/session"
)

// errNoSession signale l'absence de cookie de session.
var errNoSession = errors.New("aucune session")

// authenticatedHandler est un handler de page qui a besoin de connaître
// l'utilisateur connecté et son jeton.
type authenticatedHandler func(w http.ResponseWriter, r *http.Request, user *api.User, token string)

// requireAuth protège une page derrière l'authentification.
//
// Elle joue le même rôle que le middleware du serveur, mais avec une réponse
// adaptée à un navigateur : au lieu d'un 401 en JSON, l'utilisateur est
// redirigé vers la page de connexion.
//
// Le fait que le handler protégé reçoive user et token en paramètres, plutôt
// que de les relire lui-même, est délibéré : il ne peut pas s'exécuter sans
// eux, et il n'existe donc pas de chemin où une page protégée s'afficherait
// pour un visiteur anonyme.
func (h *Handler) requireAuth(next authenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, token, err := h.currentUser(r)
		if err != nil {
			// Session absente, ou jeton refusé par l'API (expiré,
			// invalide) : on efface le cookie devenu inutile et on renvoie
			// vers la connexion.
			if errors.Is(err, errNoSession) || api.IsUnauthorized(err) {
				session.Clear(w)
				redirectToLogin(w, r)
				return
			}

			// Autre échec : l'API est injoignable ou en défaut. Ce n'est pas
			// un problème d'authentification, il ne faut donc pas déconnecter
			// l'utilisateur pour autant.
			log.Printf("vérification de la session : %v", err)
			h.serverError(w, r, nil)
			return
		}

		next(w, r, user, token)
	}
}

// redirectToLogin renvoie vers la page de connexion.
//
// Le statut 303 (See Other) est utilisé plutôt que 302 : il impose au
// navigateur de suivre la redirection en GET, y compris lorsque la requête
// d'origine était un POST. Avec un 302, certains navigateurs rejoueraient le
// POST vers la nouvelle adresse.
func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
