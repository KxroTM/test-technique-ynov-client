package handlers

import (
	"log"
	"net/http"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
	"github.com/KxroTM/test-technique-ynov-client/internal/session"
)

// loginForm porte les valeurs du formulaire de connexion
type loginForm struct {
	Email string
}

// registerForm porte les valeurs du formulaire d'inscription
type registerForm struct {
	Email string
	Name  string
}

// spaceForm porte les valeurs du formulaire d'espace
type spaceForm struct {
	Name        string
	Description string
}

// noteForm porte les valeurs du formulaire de note
type noteForm struct {
	Title   string
	Content string
	Status  api.NoteStatus
}

// ShowLogin affiche le formulaire de connexion
func (h *Handler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	if _, _, err := h.currentUser(r); err == nil {
		http.Redirect(w, r, "/spaces", http.StatusSeeOther)
		return
	}

	data := newPageData("Connexion", nil)
	data.Flash = takeFlash(w, r)
	data.ErrorMessage = takeFlashError(w, r)
	data.GoogleEnabled = h.googleEnabled
	data.Form = loginForm{}

	h.render(w, http.StatusOK, "login.html", data)
}

// Login traite l'envoi du formulaire de connexion
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	form := loginForm{Email: r.FormValue("email")}
	password := r.FormValue("password")

	result, err := h.api.Login(r.Context(), form.Email, password)
	if err != nil {
		data := newPageData("Connexion", nil)
		data.Form = form
		data.GoogleEnabled = h.googleEnabled
		applyAPIError(&data, err)

		if _, ok := api.AsError(err); !ok {
			log.Printf("connexion : %v", err)
		}

		h.render(w, statusForFormError(err), "login.html", data)
		return
	}

	session.Save(w, result.Token, result.ExpiresAt)
	http.Redirect(w, r, "/spaces", http.StatusSeeOther)
}

// ShowRegister affiche le formulaire d'inscription
func (h *Handler) ShowRegister(w http.ResponseWriter, r *http.Request) {
	if _, _, err := h.currentUser(r); err == nil {
		http.Redirect(w, r, "/spaces", http.StatusSeeOther)
		return
	}

	data := newPageData("Créer un compte", nil)
	data.GoogleEnabled = h.googleEnabled
	data.Form = registerForm{}

	h.render(w, http.StatusOK, "register.html", data)
}

// Register traite l'envoi du formulaire d'inscription
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	form := registerForm{
		Email: r.FormValue("email"),
		Name:  r.FormValue("name"),
	}
	password := r.FormValue("password")

	if _, err := h.api.Register(r.Context(), form.Email, password, form.Name); err != nil {
		data := newPageData("Créer un compte", nil)
		data.Form = form
		data.GoogleEnabled = h.googleEnabled
		applyAPIError(&data, err)

		if _, ok := api.AsError(err); !ok {
			log.Printf("inscription : %v", err)
		}

		h.render(w, statusForFormError(err), "register.html", data)
		return
	}

	result, err := h.api.Login(r.Context(), form.Email, password)
	if err != nil {
		log.Printf("connexion automatique après inscription : %v", err)
		setFlash(w, "Votre compte a été créé. Connectez-vous pour continuer.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session.Save(w, result.Token, result.ExpiresAt)
	setFlash(w, "Bienvenue ! Créez un premier espace pour ranger vos notes.")
	http.Redirect(w, r, "/spaces", http.StatusSeeOther)
}

// Logout efface la session et renvoie vers la page de connexion
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	session.Clear(w)
	setFlash(w, "Vous êtes déconnecté.")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Home oriente le visiteur selon qu'il a une session valide ou non
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if _, _, err := h.currentUser(r); err != nil {
		redirectToLogin(w, r)
		return
	}
	http.Redirect(w, r, "/spaces", http.StatusSeeOther)
}

// statusForFormError choisit le code de statut du réaffichage d'un formulaire
func statusForFormError(err error) int {
	if apiErr, ok := api.AsError(err); ok {
		return apiErr.StatusCode
	}
	return http.StatusServiceUnavailable
}
