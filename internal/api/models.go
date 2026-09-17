// Package api contient le client HTTP qui dialogue avec le serveur.
//
// C'est la seule partie du client web qui connaît la forme de l'API : ses
// routes, ses formats JSON et ses codes de statut. Les handlers de pages
// appellent des méthodes Go typées et ne construisent jamais de requête HTTP
// eux-mêmes. Si un endpoint du serveur change, ce package est le seul à
// modifier.
package api

import "time"

// NoteStatus représente l'état d'avancement d'une note.
// Les valeurs sont celles attendues par l'API.
type NoteStatus string

const (
	StatusTodo       NoteStatus = "todo"
	StatusInProgress NoteStatus = "in_progress"
	StatusDone       NoteStatus = "done"
)

// Label retourne le libellé français de l'état, destiné à l'affichage.
// Il est utilisé directement par les templates.
func (s NoteStatus) Label() string {
	switch s {
	case StatusTodo:
		return "Non fait"
	case StatusInProgress:
		return "En cours"
	case StatusDone:
		return "Terminé"
	default:
		return string(s)
	}
}

// User représente un utilisateur tel que retourné par l'API.
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Space représente un espace tel que retourné par l'API.
type Space struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	NoteCount   int       `json:"note_count"`
}

// Note représente une note telle que retournée par l'API.
type Note struct {
	ID        int64      `json:"id"`
	SpaceID   int64      `json:"space_id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Status    NoteStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// SpaceWithNotes est la réponse de la consultation d'un espace : l'espace
// lui-même accompagné de ses notes.
type SpaceWithNotes struct {
	Space *Space `json:"space"`
	Notes []Note `json:"notes"`
}

// LoginResult est la réponse d'une connexion réussie.
type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *User     `json:"user"`
}

// AllStatuses retourne les états sélectionnables dans les formulaires.
//
// La liste est définie ici plutôt que dans les templates : ajouter un état
// ne nécessitera pas de retoucher le HTML.
func AllStatuses() []NoteStatus {
	return []NoteStatus{StatusTodo, StatusInProgress, StatusDone}
}
