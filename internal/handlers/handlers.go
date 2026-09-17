// Package handlers contient les handlers de pages du client web.
//
// Chaque handler suit la même séquence : lire la session, appeler l'API via le
// client typé, puis rendre un gabarit ou rediriger. Aucune règle métier n'est
// écrite ici : elle appartient au serveur. Le client ne fait que présenter les
// données et transmettre les intentions de l'utilisateur.
package handlers

import (
	"log"
	"net/http"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
	"github.com/KxroTM/test-technique-ynov-client/internal/render"
	"github.com/KxroTM/test-technique-ynov-client/internal/session"
)

// Handler porte les dépendances partagées par tous les handlers.
type Handler struct {
	api      *api.Client
	renderer *render.Renderer
}

// New construit le handler.
func New(apiClient *api.Client, renderer *render.Renderer) *Handler {
	return &Handler{api: apiClient, renderer: renderer}
}

// pageData est la structure passée à tous les gabarits.
//
// Une structure unique, plutôt qu'une map par page, permet au compilateur de
// détecter les champs mal nommés côté Go. Les gabarits, eux, tolèrent les
// champs absents : une page qui n'affiche pas de formulaire laisse simplement
// Form à sa valeur nulle.
type pageData struct {
	Title string
	User  *api.User

	// Flash est un message de succès affiché une seule fois, après une
	// redirection.
	Flash string

	// ErrorMessage est un message d'erreur global, affiché en bandeau.
	ErrorMessage string

	// FieldErrors associe un nom de champ de formulaire à son message
	// d'erreur. Il est toujours initialisé, afin que les gabarits puissent
	// appeler `index .FieldErrors "email"` sans condition préalable.
	FieldErrors map[string]string

	// Form porte les valeurs saisies, réaffichées après une erreur pour que
	// l'utilisateur n'ait pas à tout retaper.
	Form any

	// FormAction, SubmitLabel et CancelURL permettent à un même gabarit de
	// servir la création et la modification.
	FormAction  string
	SubmitLabel string
	CancelURL   string

	// Données métier, selon la page.
	Spaces   []api.Space
	Space    *api.Space
	Notes    []api.Note
	Statuses []api.NoteStatus

	// Champs propres à la page d'erreur.
	StatusCode int
	Heading    string
	Detail     string
}

// newPageData initialise les données communes d'une page.
func newPageData(title string, user *api.User) pageData {
	return pageData{
		Title:       title,
		User:        user,
		FieldErrors: map[string]string{},
	}
}

// render rend une page et journalise l'échec éventuel.
//
// Une erreur de rendu ne peut plus être présentée à l'utilisateur de façon
// fiable : soit la réponse est déjà partie, soit le gabarit lui-même est en
// cause. On la journalise donc pour pouvoir la corriger, et on s'arrête là.
func (h *Handler) render(w http.ResponseWriter, statusCode int, page string, data pageData) {
	if err := h.renderer.Page(w, statusCode, page, data); err != nil {
		log.Printf("rendu de %s : %v", page, err)
	}
}

// applyAPIError répartit une erreur de l'API dans les données de page.
//
// Les erreurs de validation sont placées champ par champ afin que le gabarit
// les affiche sous les bons champs ; le message global sert de repli quand
// l'API n'a pas détaillé.
func applyAPIError(data *pageData, err error) {
	apiErr, ok := api.AsError(err)
	if !ok {
		// Erreur de transport : l'API est injoignable. Le détail technique
		// est journalisé par l'appelant, l'utilisateur reçoit un message
		// compréhensible.
		data.ErrorMessage = "Le service est momentanément indisponible. Réessayez dans un instant."
		return
	}

	data.ErrorMessage = apiErr.Message

	for field, message := range apiErr.Fields {
		data.FieldErrors[field] = message
	}
}

// currentUser retourne l'utilisateur authentifié d'après la session.
//
// Le profil est demandé à l'API à chaque requête plutôt que stocké dans le
// cookie. C'est un appel supplémentaire, mais il garantit que le jeton est
// toujours valide et que le nom affiché est à jour. Sur une application de
// cette taille, la simplicité vaut mieux que l'économie d'une requête.
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
