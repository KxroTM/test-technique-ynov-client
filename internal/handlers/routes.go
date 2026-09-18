package handlers

import "net/http"

// Routes construit le routeur du client web
func (h *Handler) Routes(staticHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// Fichiers statiques
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticHandler))

	// Pages publiques
	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /login", h.ShowLogin)
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("GET /register", h.ShowRegister)
	mux.HandleFunc("POST /register", h.Register)
	mux.HandleFunc("POST /logout", h.Logout)
	mux.HandleFunc("GET /auth/google", h.StartGoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", h.CompleteGoogleLogin)

	// Espaces
	mux.HandleFunc("GET /spaces", h.requireAuth(h.ListSpaces))
	mux.HandleFunc("GET /spaces/new", h.requireAuth(h.NewSpace))
	mux.HandleFunc("POST /spaces/new", h.requireAuth(h.CreateSpace))
	mux.HandleFunc("GET /spaces/{spaceID}", h.requireAuth(h.ShowSpace))
	mux.HandleFunc("GET /spaces/{spaceID}/edit", h.requireAuth(h.EditSpace))
	mux.HandleFunc("POST /spaces/{spaceID}/edit", h.requireAuth(h.UpdateSpace))
	mux.HandleFunc("POST /spaces/{spaceID}/delete", h.requireAuth(h.DeleteSpace))

	// Notes
	mux.HandleFunc("GET /spaces/{spaceID}/notes/new", h.requireAuth(h.NewNote))
	mux.HandleFunc("POST /spaces/{spaceID}/notes/new", h.requireAuth(h.CreateNote))
	mux.HandleFunc("GET /notes/{noteID}/edit", h.requireAuth(h.EditNote))
	mux.HandleFunc("POST /notes/{noteID}/edit", h.requireAuth(h.UpdateNote))
	mux.HandleFunc("POST /notes/{noteID}/status", h.requireAuth(h.UpdateNoteStatus))
	mux.HandleFunc("POST /notes/{noteID}/delete", h.requireAuth(h.DeleteNote))

	// 404
	mux.HandleFunc("/", h.NotFoundPage)

	return mux
}
