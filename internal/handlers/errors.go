package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
)

// notFound affiche la page « ressource introuvable ».
//
// Elle est utilisée aussi bien pour une URL inconnue que pour une ressource
// appartenant à un autre utilisateur : le serveur ne distingue pas les deux
// cas, et le client ne cherche pas à le faire non plus.
func (h *Handler) notFound(w http.ResponseWriter, _ *http.Request, user *api.User) {
	data := newPageData("Page introuvable", user)
	data.StatusCode = http.StatusNotFound
	data.Heading = "Cette page n'existe pas"
	data.Detail = "Le lien est peut-être erroné, ou la ressource a été supprimée."

	h.render(w, http.StatusNotFound, "error.html", data)
}

// serverError affiche la page d'erreur interne.
//
// Le détail technique n'est jamais transmis au navigateur : il est journalisé
// côté serveur. L'utilisateur ne peut rien en faire, et il renseignerait un
// éventuel attaquant sur l'infrastructure.
func (h *Handler) serverError(w http.ResponseWriter, _ *http.Request, user *api.User) {
	data := newPageData("Erreur", user)
	data.StatusCode = http.StatusInternalServerError
	data.Heading = "Une erreur est survenue"
	data.Detail = "Le service n'a pas pu traiter votre demande. Réessayez dans un instant."

	h.render(w, http.StatusInternalServerError, "error.html", data)
}

// handleAPIError choisit la page d'erreur correspondant à une erreur de l'API.
//
// Elle regroupe le traitement commun à toutes les pages de consultation :
// ressource introuvable, session expirée, ou échec inattendu. Les erreurs de
// validation ne passent pas par ici — elles doivent réafficher le formulaire,
// pas une page d'erreur.
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

// pathID lit un identifiant numérique depuis un paramètre d'URL.
//
// Les identifiants invalides sont traités comme des ressources introuvables :
// pour un visiteur, une URL qui ne désigne rien et une URL malformée sont la
// même chose. L'API, elle, distingue les deux cas avec un 400, ce qui a du
// sens pour un client programmatique.
func pathID(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// NotFoundPage rend la page « introuvable » pour une adresse inconnue.
//
// L'utilisateur est résolu au mieux : s'il a une session valide, l'en-tête
// affiche son nom et le lien de déconnexion, ce qui évite de donner
// l'impression d'avoir été déconnecté par une simple faute de frappe dans
// l'URL.
func (h *Handler) NotFoundPage(w http.ResponseWriter, r *http.Request) {
	user, _, err := h.currentUser(r)
	if err != nil {
		user = nil
	}

	h.notFound(w, r, user)
}
