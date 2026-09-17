package api

import (
	"errors"
	"fmt"
	"net/http"
)

// Error représente une réponse d'erreur du serveur.
//
// Elle conserve le code de statut HTTP et, le cas échéant, le détail des
// erreurs de validation champ par champ. Les handlers de pages s'en servent
// pour réafficher un formulaire avec les messages placés sous les bons
// champs, plutôt qu'un message global peu exploitable.
type Error struct {
	StatusCode int
	Message    string
	Fields     map[string]string
}

// Error implémente l'interface error.
func (e *Error) Error() string {
	return fmt.Sprintf("api: %d %s", e.StatusCode, e.Message)
}

// IsUnauthorized indique que le jeton est absent, invalide ou expiré, ou que
// les identifiants fournis sont incorrects.
func (e *Error) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsNotFound indique que la ressource n'existe pas ou n'appartient pas à
// l'utilisateur connecté. Le serveur ne distingue volontairement pas les deux
// cas, et le client ne peut donc pas le faire non plus.
func (e *Error) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsValidation indique une erreur de saisie de l'utilisateur : données
// invalides (400) ou conflit avec une ressource existante (409). Dans les deux
// cas, le formulaire doit être réaffiché avec le message, et non remplacé par
// une page d'erreur.
func (e *Error) IsValidation() bool {
	return e.StatusCode == http.StatusBadRequest || e.StatusCode == http.StatusConflict
}

// FieldError retourne le message d'erreur associé à un champ, s'il existe.
// Elle est appelée depuis les templates pour afficher l'erreur sous le champ.
func (e *Error) FieldError(field string) string {
	if e == nil || e.Fields == nil {
		return ""
	}
	return e.Fields[field]
}

// AsError extrait une *Error d'une erreur quelconque.
//
// Elle évite de répéter le même errors.As dans chaque handler.
func AsError(err error) (*Error, bool) {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// IsUnauthorized indique si une erreur quelconque correspond à une session
// invalide. C'est le cas qui déclenche une redirection vers la page de
// connexion.
func IsUnauthorized(err error) bool {
	apiErr, ok := AsError(err)
	return ok && apiErr.IsUnauthorized()
}

// IsNotFound indique si une erreur quelconque correspond à une ressource
// introuvable ou inaccessible.
func IsNotFound(err error) bool {
	apiErr, ok := AsError(err)
	return ok && apiErr.IsNotFound()
}
