package handlers

import "net/http"

// Routes construit le routeur du client web.
//
// Le routeur est celui de la bibliothèque standard. Depuis Go 1.22, net/http
// sait associer une méthode et des paramètres d'URL à un handler
// (« POST /spaces/{spaceID}/delete »), ce qui rend toute dépendance à un
// routeur tiers superflue pour une application de cette taille.
//
// Deux partis pris sont visibles dans cette table de routes.
//
// D'abord, les modifications passent toutes par POST. Un formulaire HTML ne
// sait émettre que GET et POST : les verbes PUT et DELETE lui sont
// inaccessibles. Le client expose donc des routes POST explicites
// (« /spaces/{id}/delete »), et c'est le client API qui les traduit en PUT et
// DELETE vers le serveur. L'API REST reste ainsi correcte, et l'application
// fonctionne sans une ligne de JavaScript.
//
// Ensuite, l'affichage d'un formulaire et son traitement partagent la même
// adresse, distingués par la méthode : GET affiche, POST enregistre. L'URL
// visible dans la barre d'adresse reste donc la même en cas d'erreur de
// saisie.
func (h *Handler) Routes(staticHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	// Fichiers statiques (feuille de style).
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticHandler))

	// Pages publiques.
	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /login", h.ShowLogin)
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("GET /register", h.ShowRegister)
	mux.HandleFunc("POST /register", h.Register)
	mux.HandleFunc("POST /logout", h.Logout)

	// Espaces. Chaque page est enveloppée par requireAuth : la protection
	// est visible ici, route par route, et un oubli se repère à la lecture.
	mux.HandleFunc("GET /spaces", h.requireAuth(h.ListSpaces))
	mux.HandleFunc("GET /spaces/new", h.requireAuth(h.NewSpace))
	mux.HandleFunc("POST /spaces/new", h.requireAuth(h.CreateSpace))
	mux.HandleFunc("GET /spaces/{spaceID}", h.requireAuth(h.ShowSpace))
	mux.HandleFunc("GET /spaces/{spaceID}/edit", h.requireAuth(h.EditSpace))
	mux.HandleFunc("POST /spaces/{spaceID}/edit", h.requireAuth(h.UpdateSpace))
	mux.HandleFunc("POST /spaces/{spaceID}/delete", h.requireAuth(h.DeleteSpace))

	// Notes.
	mux.HandleFunc("GET /spaces/{spaceID}/notes/new", h.requireAuth(h.NewNote))
	mux.HandleFunc("POST /spaces/{spaceID}/notes/new", h.requireAuth(h.CreateNote))
	mux.HandleFunc("GET /notes/{noteID}/edit", h.requireAuth(h.EditNote))
	mux.HandleFunc("POST /notes/{noteID}/edit", h.requireAuth(h.UpdateNote))
	mux.HandleFunc("POST /notes/{noteID}/delete", h.requireAuth(h.DeleteNote))

	// Route attrape-tout : toute adresse non reconnue ci-dessus atterrit ici.
	// Sans elle, le routeur répondrait le « 404 page not found » brut de la
	// bibliothèque standard, sans mise en page ni navigation pour revenir.
	// Le motif « / » est le moins spécifique : il ne s'applique qu'en dernier
	// recours.
	mux.HandleFunc("/", h.NotFoundPage)

	return mux
}
