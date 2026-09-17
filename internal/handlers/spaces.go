package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
)

// ListSpaces affiche la liste des espaces de l'utilisateur.
func (h *Handler) ListSpaces(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	spaces, err := h.api.ListSpaces(r.Context(), token)
	if err != nil {
		h.handleAPIError(w, r, user, err)
		return
	}

	data := newPageData("Mes espaces", user)
	data.Flash = takeFlash(w, r)
	data.Spaces = spaces

	h.render(w, http.StatusOK, "spaces.html", data)
}

// ShowSpace affiche un espace et ses notes.
func (h *Handler) ShowSpace(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	spaceID, ok := pathID(r, "spaceID")
	if !ok {
		h.notFound(w, r, user)
		return
	}

	// Un seul appel retourne l'espace et ses notes : l'API les renvoie
	// ensemble parce que cette page a besoin des deux.
	result, err := h.api.ListSpaceNotes(r.Context(), token, spaceID)
	if err != nil {
		h.handleAPIError(w, r, user, err)
		return
	}

	data := newPageData(result.Space.Name, user)
	data.Flash = takeFlash(w, r)
	data.Space = result.Space
	data.Notes = result.Notes

	h.render(w, http.StatusOK, "space_detail.html", data)
}

// NewSpace affiche le formulaire de création d'un espace.
func (h *Handler) NewSpace(w http.ResponseWriter, r *http.Request, user *api.User, _ string) {
	data := newPageData("Nouvel espace", user)
	data.Form = spaceForm{}
	data.FormAction = "/spaces/new"
	data.SubmitLabel = "Créer l'espace"
	data.CancelURL = "/spaces"

	h.render(w, http.StatusOK, "space_form.html", data)
}

// CreateSpace traite l'envoi du formulaire de création.
func (h *Handler) CreateSpace(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	form := spaceForm{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}

	space, err := h.api.CreateSpace(r.Context(), token, form.Name, form.Description)
	if err != nil {
		h.renderSpaceFormError(w, r, user, err, form, "Nouvel espace", "/spaces/new", "Créer l'espace", "/spaces")
		return
	}

	setFlash(w, fmt.Sprintf("L'espace « %s » a été créé.", space.Name))
	http.Redirect(w, r, fmt.Sprintf("/spaces/%d", space.ID), http.StatusSeeOther)
}

// EditSpace affiche le formulaire de modification d'un espace.
//
// L'espace est relu depuis l'API pour pré-remplir le formulaire. Cette lecture
// sert aussi de contrôle d'accès : si l'espace n'appartient pas à
// l'utilisateur, l'API répond 404 et le formulaire ne s'affiche jamais.
func (h *Handler) EditSpace(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	spaceID, ok := pathID(r, "spaceID")
	if !ok {
		h.notFound(w, r, user)
		return
	}

	space, err := h.api.GetSpace(r.Context(), token, spaceID)
	if err != nil {
		h.handleAPIError(w, r, user, err)
		return
	}

	data := newPageData("Modifier l'espace", user)
	data.Space = space
	data.Form = spaceForm{Name: space.Name, Description: space.Description}
	data.FormAction = fmt.Sprintf("/spaces/%d/edit", space.ID)
	data.SubmitLabel = "Enregistrer"
	data.CancelURL = fmt.Sprintf("/spaces/%d", space.ID)

	h.render(w, http.StatusOK, "space_form.html", data)
}

// UpdateSpace traite l'envoi du formulaire de modification.
func (h *Handler) UpdateSpace(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	spaceID, ok := pathID(r, "spaceID")
	if !ok {
		h.notFound(w, r, user)
		return
	}

	form := spaceForm{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}

	space, err := h.api.UpdateSpace(r.Context(), token, spaceID, form.Name, form.Description)
	if err != nil {
		action := fmt.Sprintf("/spaces/%d/edit", spaceID)
		cancel := fmt.Sprintf("/spaces/%d", spaceID)
		h.renderSpaceFormError(w, r, user, err, form, "Modifier l'espace", action, "Enregistrer", cancel)
		return
	}

	setFlash(w, "L'espace a été mis à jour.")
	http.Redirect(w, r, fmt.Sprintf("/spaces/%d", space.ID), http.StatusSeeOther)
}

// DeleteSpace supprime un espace et ses notes.
func (h *Handler) DeleteSpace(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	spaceID, ok := pathID(r, "spaceID")
	if !ok {
		h.notFound(w, r, user)
		return
	}

	if err := h.api.DeleteSpace(r.Context(), token, spaceID); err != nil {
		h.handleAPIError(w, r, user, err)
		return
	}

	setFlash(w, "L'espace et ses notes ont été supprimés.")
	http.Redirect(w, r, "/spaces", http.StatusSeeOther)
}

// renderSpaceFormError réaffiche le formulaire d'espace après un échec.
//
// La création et la modification partagent ce traitement : seules l'action du
// formulaire et les libellés changent. Les erreurs de validation réaffichent
// le formulaire ; une ressource introuvable ou une panne relèvent en revanche
// d'une page d'erreur.
func (h *Handler) renderSpaceFormError(
	w http.ResponseWriter,
	r *http.Request,
	user *api.User,
	err error,
	form spaceForm,
	title, action, submitLabel, cancelURL string,
) {
	if apiErr, ok := api.AsError(err); ok && !apiErr.IsValidation() {
		h.handleAPIError(w, r, user, err)
		return
	}

	if _, ok := api.AsError(err); !ok {
		log.Printf("enregistrement de l'espace : %v", err)
	}

	data := newPageData(title, user)
	data.Form = form
	data.FormAction = action
	data.SubmitLabel = submitLabel
	data.CancelURL = cancelURL
	applyAPIError(&data, err)

	h.render(w, statusForFormError(err), "space_form.html", data)
}
