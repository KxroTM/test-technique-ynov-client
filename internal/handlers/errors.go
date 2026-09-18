package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
)

// notFound affiche la page "ressource introuvable"
func (h *Handler) notFound(w http.ResponseWriter, _ *http.Request, user *api.User) {
	data := newPageData("Page introuvable", user)
	data.StatusCode = http.StatusNotFound
	data.Heading = "Cette page n'existe pas"
	data.Detail = "Le lien est peut-être erroné, ou la ressource a été supprimée."

	h.render(w, http.StatusNotFound, "error.html", data)
}

// serverError affiche la page d'erreur interne
func (h *Handler) serverError(w http.ResponseWriter, _ *http.Request, user *api.User) {
	data := newPageData("Erreur", user)
	data.StatusCode = http.StatusInternalServerError
	data.Heading = "Une erreur est survenue"
	data.Detail = "Le service n'a pas pu traiter votre demande. Réessayez dans un instant."

	h.render(w, http.StatusInternalServerError, "error.html", data)
}

// handleAPIError choisit la page d'erreur correspondant à une erreur de l'API
func (h *Handler) handleAPIError(w http.ResponseWriter, r *http.Request, user *api.User, err error) {
	switch {
	case api.IsNotFound(err):
		h.notFound(w, r, user)

	case api.IsUnauthorized(err):
		redirectToLogin(w, r)

	default:
		log.Printf("appel à l'API : %v", err)
		h.serverError(w, r, user)
	}
}

// pathID lit un identifiant numérique depuis un paramètre d'URL
func pathID(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// NotFoundPage rend la page « introuvable » pour une adresse inconnue
func (h *Handler) NotFoundPage(w http.ResponseWriter, r *http.Request) {
	user, _, err := h.currentUser(r)
	if err != nil {
		user = nil
	}

	h.notFound(w, r, user)
}
