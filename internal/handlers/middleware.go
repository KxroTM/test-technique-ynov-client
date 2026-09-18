package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
	"github.com/KxroTM/test-technique-ynov-client/internal/session"
)

// errNoSession signale l'absence de cookie de session
var errNoSession = errors.New("aucune session")

// authenticatedHandler est un handler de page qui a besoin de connaître l'utilisateur connecté et son jeton
type authenticatedHandler func(w http.ResponseWriter, r *http.Request, user *api.User, token string)

// requireAuth protège une page derrière l'authentification
func (h *Handler) requireAuth(next authenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, token, err := h.currentUser(r)
		if err != nil {

			if errors.Is(err, errNoSession) || api.IsUnauthorized(err) {
				session.Clear(w)
				redirectToLogin(w, r)
				return
			}

			log.Printf("vérification de la session : %v", err)
			h.serverError(w, r, nil)
			return
		}

		next(w, r, user, token)
	}
}

// redirectToLogin renvoie vers la page de connexion
func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
