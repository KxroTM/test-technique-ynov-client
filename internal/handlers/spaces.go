package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
)

// ListSpaces affiche la liste des espaces de l'utilisateur
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

// ShowSpace affiche un espace et ses notes
func (h *Handler) ShowSpace(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	spaceID, ok := pathID(r, "spaceID")
	if !ok {
		h.notFound(w, r, user)
		return
	}

	result, err := h.api.ListSpaceNotes(r.Context(), token, spaceID)
	if err != nil {
		h.handleAPIError(w, r, user, err)
		return
	}

	data := newPageData(result.Space.Name, user)
	data.Flash = takeFlash(w, r)
	data.Space = result.Space
	data.Notes = result.Notes
	data.Board = buildBoard(result.Notes)
	data.Progress = donePercent(data.Board)

	h.render(w, http.StatusOK, "space_detail.html", data)
}

// NewSpace affiche le formulaire de création d'un espace
func (h *Handler) NewSpace(w http.ResponseWriter, r *http.Request, user *api.User, _ string) {
	data := newPageData("Nouvel espace", user)
	data.Form = spaceForm{}
	data.FormAction = "/spaces/new"
	data.SubmitLabel = "Créer l'espace"
	data.CancelURL = "/spaces"

	h.render(w, http.StatusOK, "space_form.html", data)
}

// CreateSpace traite l'envoi du formulaire de création
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

// EditSpace affiche le formulaire de modification d'un espace
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

// UpdateSpace traite l'envoi du formulaire de modification
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

// DeleteSpace supprime un espace et ses notes
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

// renderSpaceFormError réaffiche le formulaire d'espace après un échec
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

// boardColumn regroupe les notes d'un même état pour l'affichage en colonnes
type boardColumn struct {
	Status  api.NoteStatus
	Notes   []api.Note
	Percent int
}

// buildBoard répartit les notes d'un espace dans une colonne par état et calcule leur part du total
func buildBoard(notes []api.Note) []boardColumn {
	statuses := api.AllStatuses()
	columns := make([]boardColumn, 0, len(statuses))

	for _, status := range statuses {
		column := boardColumn{Status: status, Notes: []api.Note{}}
		for _, note := range notes {
			if note.Status == status {
				column.Notes = append(column.Notes, note)
			}
		}
		if len(notes) > 0 {
			column.Percent = len(column.Notes) * 100 / len(notes)
		}
		columns = append(columns, column)
	}

	return columns
}

// donePercent retourne la part de notes terminées pour la jauge
func donePercent(columns []boardColumn) int {
	for _, column := range columns {
		if column.Status == api.StatusDone {
			return column.Percent
		}
	}
	return 0
}
