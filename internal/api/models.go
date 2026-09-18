// Package api contient le client HTTP qui dialogue avec le serveur

package api

import (
	"strings"
	"time"
	"unicode"
)

// NoteStatus représente l'état d'avancement d'une note
type NoteStatus string

const (
	StatusTodo       NoteStatus = "todo"
	StatusInProgress NoteStatus = "in_progress"
	StatusDone       NoteStatus = "done"
)

// IsValid indique si l'état fait partie des valeurs acceptées par l'API
func (s NoteStatus) IsValid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

// Prev retourne l'état qui précède dans la progression, ou l'état lui-même s'il est déjà le premier
func (s NoteStatus) Prev() NoteStatus {
	switch s {
	case StatusDone:
		return StatusInProgress
	case StatusInProgress:
		return StatusTodo
	default:
		return s
	}
}

// Next retourne l'état qui suit dans la progression, ou l'état lui-même s'il est déjà le dernier
func (s NoteStatus) Next() NoteStatus {
	switch s {
	case StatusTodo:
		return StatusInProgress
	case StatusInProgress:
		return StatusDone
	default:
		return s
	}
}

// Label retourne le libellé français de l'état
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

// User représente un utilisateur tel que retourné par l'API
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Space représente un espace tel que retourné par l'API
type Space struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	NoteCount   int       `json:"note_count"`
}

// Note représente une note telle que retournée par l'API
type Note struct {
	ID        int64      `json:"id"`
	SpaceID   int64      `json:"space_id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Status    NoteStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// SpaceWithNotes est la réponse de la consultation d'un espace
type SpaceWithNotes struct {
	Space *Space `json:"space"`
	Notes []Note `json:"notes"`
}

// LoginResult est la réponse d'une connexion réussie
type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *User     `json:"user"`
}

// AllStatuses retourne les états sélectionnables dans les formulaires
func AllStatuses() []NoteStatus {
	return []NoteStatus{StatusTodo, StatusInProgress, StatusDone}
}

// paletteCount est le nombre de teintes disponibles pour identifier un espace
const paletteCount = 8

// Palette retourne la teinte d'un espace, dérivée de son nom pour rester stable d'un affichage à l'autre
func (s Space) Palette() int {
	var hash uint32 = 2166136261
	for _, r := range strings.ToLower(strings.TrimSpace(s.Name)) {
		hash ^= uint32(r)
		hash *= 16777619
	}
	return int(hash%paletteCount) + 1
}

// Initial retourne la première lettre du nom de l'espace
func (s Space) Initial() string {
	for _, r := range strings.TrimSpace(s.Name) {
		return string(unicode.ToUpper(r))
	}
	return "?"
}
