package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
)

// NewNote affiche le formulaire d'ajout d'une note dans un espace
func (h *Handler) NewNote(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
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

	data := newPageData("Nouvelle note", user)
	data.Space = space
	data.Statuses = api.AllStatuses()
	data.Form = noteForm{Status: statusFromQuery(r)}
	data.FormAction = fmt.Sprintf("/spaces/%d/notes/new", space.ID)
	data.SubmitLabel = "Ajouter la note"
	data.CancelURL = fmt.Sprintf("/spaces/%d", space.ID)

	h.render(w, http.StatusOK, "note_form.html", data)
}

// CreateNote traite l'envoi du formulaire d'ajout
func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	spaceID, ok := pathID(r, "spaceID")
	if !ok {
		h.notFound(w, r, user)
		return
	}

	form := noteForm{
		Title:   r.FormValue("title"),
		Content: r.FormValue("content"),
		Status:  api.NoteStatus(r.FormValue("status")),
	}

	if _, err := h.api.CreateNote(r.Context(), token, spaceID, form.Title, form.Content, form.Status); err != nil {
		action := fmt.Sprintf("/spaces/%d/notes/new", spaceID)
		h.renderNoteFormError(w, r, user, token, spaceID, err, form, "Nouvelle note", action, "Ajouter la note")
		return
	}

	setFlash(w, "La note a été ajoutée.")
	http.Redirect(w, r, fmt.Sprintf("/spaces/%d", spaceID), http.StatusSeeOther)
}

// EditNote affiche le formulaire de modification d'une note
func (h *Handler) EditNote(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	noteID, ok := pathID(r, "noteID")
	if !ok {
		h.notFound(w, r, user)
		return
	}

	note, err := h.api.GetNote(r.Context(), token, noteID)
	if err != nil {
		h.handleAPIError(w, r, user, err)
		return
	}

	space, err := h.api.GetSpace(r.Context(), token, note.SpaceID)
	if err != nil {
		h.handleAPIError(w, r, user, err)
		return
	}

	data := newPageData("Modifier la note", user)
	data.Space = space
	data.Statuses = api.AllStatuses()
	data.Form = noteForm{Title: note.Title, Content: note.Content, Status: note.Status}
	data.FormAction = fmt.Sprintf("/notes/%d/edit", note.ID)
	data.SubmitLabel = "Enregistrer"
	data.CancelURL = fmt.Sprintf("/spaces/%d", note.SpaceID)

	h.render(w, http.StatusOK, "note_form.html", data)
}

// UpdateNote traite l'envoi du formulaire de modification
func (h *Handler) UpdateNote(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	noteID, ok := pathID(r, "noteID")
	if !ok {
		h.notFound(w, r, user)
		return
	}

	form := noteForm{
		Title:   r.FormValue("title"),
		Content: r.FormValue("content"),
		Status:  api.NoteStatus(r.FormValue("status")),
	}

	note, err := h.api.UpdateNote(r.Context(), token, noteID, form.Title, form.Content, form.Status)
	if err != nil {
		spaceID := int64(0)
		if existing, readErr := h.api.GetNote(r.Context(), token, noteID); readErr == nil {
			spaceID = existing.SpaceID
		}

		action := fmt.Sprintf("/notes/%d/edit", noteID)
		h.renderNoteFormError(w, r, user, token, spaceID, err, form, "Modifier la note", action, "Enregistrer")
		return
	}

	setFlash(w, "La note a été mise à jour.")
	http.Redirect(w, r, fmt.Sprintf("/spaces/%d", note.SpaceID), http.StatusSeeOther)
}

// DeleteNote supprime une note
func (h *Handler) DeleteNote(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	noteID, ok := pathID(r, "noteID")
	if !ok {
		h.notFound(w, r, user)
		return
	}

	note, err := h.api.GetNote(r.Context(), token, noteID)
	if err != nil {
		h.handleAPIError(w, r, user, err)
		return
	}

	if err := h.api.DeleteNote(r.Context(), token, noteID); err != nil {
		h.handleAPIError(w, r, user, err)
		return
	}

	setFlash(w, "La note a été supprimée.")
	http.Redirect(w, r, fmt.Sprintf("/spaces/%d", note.SpaceID), http.StatusSeeOther)
}

// renderNoteFormError réaffiche le formulaire de note après un échec
func (h *Handler) renderNoteFormError(
	w http.ResponseWriter,
	r *http.Request,
	user *api.User,
	token string,
	spaceID int64,
	err error,
	form noteForm,
	title, action, submitLabel string,
) {
	if apiErr, ok := api.AsError(err); ok && !apiErr.IsValidation() {
		h.handleAPIError(w, r, user, err)
		return
	}

	if _, ok := api.AsError(err); !ok {
		log.Printf("enregistrement de la note : %v", err)
	}

	space, readErr := h.api.GetSpace(r.Context(), token, spaceID)
	if readErr != nil {
		h.handleAPIError(w, r, user, readErr)
		return
	}

	data := newPageData(title, user)
	data.Space = space
	data.Statuses = api.AllStatuses()
	data.Form = form
	data.FormAction = action
	data.SubmitLabel = submitLabel
	data.CancelURL = fmt.Sprintf("/spaces/%d", spaceID)
	applyAPIError(&data, err)

	h.render(w, statusForFormError(err), "note_form.html", data)
}

// UpdateNoteStatus change uniquement l'état d'une note, sans toucher à son titre ni à son contenu
func (h *Handler) UpdateNoteStatus(w http.ResponseWriter, r *http.Request, user *api.User, token string) {
	noteID, ok := pathID(r, "noteID")
	if !ok {
		h.notFound(w, r, user)
		return
	}

	status := api.NoteStatus(r.FormValue("status"))
	if !status.IsValid() {
		h.notFound(w, r, user)
		return
	}

	note, err := h.api.GetNote(r.Context(), token, noteID)
	if err != nil {
		h.handleAPIError(w, r, user, err)
		return
	}

	if _, err := h.api.UpdateNote(r.Context(), token, noteID, note.Title, note.Content, status); err != nil {
		if isBackgroundRequest(r) {
			http.Error(w, "déplacement impossible", statusForFormError(err))
			return
		}
		h.handleAPIError(w, r, user, err)
		return
	}

	if isBackgroundRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/spaces/%d", note.SpaceID), http.StatusSeeOther)
}

// statusFromQuery lit l'état pré-sélectionné passé en paramètre d'URL par une colonne du board
func statusFromQuery(r *http.Request) api.NoteStatus {
	status := api.NoteStatus(r.URL.Query().Get("status"))
	if !status.IsValid() {
		return api.StatusTodo
	}
	return status
}

// isBackgroundRequest distingue un appel fetch du script de board d'une soumission de formulaire classique
func isBackgroundRequest(r *http.Request) bool {
	return r.Header.Get("X-Board-Request") == "1"
}
