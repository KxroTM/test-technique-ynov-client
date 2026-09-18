// Package handlers contient les handlers de pages du client web

package handlers

import (
	"log"
	"net/http"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
	"github.com/KxroTM/test-technique-ynov-client/internal/render"
	"github.com/KxroTM/test-technique-ynov-client/internal/session"
)

// Handler porte les dépendances partagées par tous les handlers
type Handler struct {
	api            *api.Client
	renderer       *render.Renderer
	googleEnabled  bool
	googleClientID string
}

// New construit le handler
func New(apiClient *api.Client, renderer *render.Renderer, googleClientID string) *Handler {
	return &Handler{
		api:            apiClient,
		renderer:       renderer,
		googleEnabled:  googleClientID != "",
		googleClientID: googleClientID,
	}
}

// pageData est la structure passée à tous les gabarits
type pageData struct {
	Title string
	User  *api.User

	// message de succès affiché après une redirection
	Flash string

	// ErrorMessage est un message d'erreur global
	ErrorMessage string

	// FieldErrors associe un nom de champ de formulaire à son message d'erreur
	FieldErrors map[string]string

	// GoogleEnabled commande l'affichage du bouton de connexion Google
	GoogleEnabled bool

	// Form porte les valeurs saisies après une erreur pour que l'utilisateur n'ait pas à tout retaper
	Form any

	// FormAction, SubmitLabel et CancelURL permettent à un même gabarit de servir la création et la modification
	FormAction  string
	SubmitLabel string
	CancelURL   string

	// Données métier
	Spaces   []api.Space
	Space    *api.Space
	Notes    []api.Note
	Statuses []api.NoteStatus

	// Board porte les notes réparties en colonnes, une par état, et Progress la part de notes terminées
	Board    []boardColumn
	Progress int

	// Champs propres à la page d'erreur
	StatusCode int
	Heading    string
	Detail     string
}

// newPageData initialise les données communes d'une page
func newPageData(title string, user *api.User) pageData {
	return pageData{
		Title:       title,
		User:        user,
		FieldErrors: map[string]string{},
	}
}

// render rend une page et journalise l'échec éventuel
func (h *Handler) render(w http.ResponseWriter, statusCode int, page string, data pageData) {
	if err := h.renderer.Page(w, statusCode, page, data); err != nil {
		log.Printf("rendu de %s : %v", page, err)
	}
}

// applyAPIError répartit une erreur de l'API dans les données de page
func applyAPIError(data *pageData, err error) {
	apiErr, ok := api.AsError(err)
	if !ok {
		data.ErrorMessage = "Le service est momentanément indisponible. Réessayez dans un instant."
		return
	}

	data.ErrorMessage = apiErr.Message

	for field, message := range apiErr.Fields {
		data.FieldErrors[field] = message
	}
}

// currentUser retourne l'utilisateur authentifié d'après la session
func (h *Handler) currentUser(r *http.Request) (*api.User, string, error) {
	token, ok := session.Token(r)
	if !ok {
		return nil, "", errNoSession
	}

	user, err := h.api.Me(r.Context(), token)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}
