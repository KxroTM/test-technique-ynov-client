package handlers

import (
	"log"
	"net/http"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
	"github.com/KxroTM/test-technique-ynov-client/internal/session"
)

// loginForm porte les valeurs du formulaire de connexion.
//
// Le mot de passe n'y figure pas : il n'est jamais réaffiché après un échec.
// Le renvoyer dans le HTML le ferait apparaître dans le cache du navigateur et
// dans l'historique de la page.
type loginForm struct {
	Email string
}

// registerForm porte les valeurs du formulaire d'inscription.
type registerForm struct {
	Email string
	Name  string
}

// spaceForm porte les valeurs du formulaire d'espace.
type spaceForm struct {
	Name        string
	Description string
}

// noteForm porte les valeurs du formulaire de note.
type noteForm struct {
	Title   string
	Content string
	Status  api.NoteStatus
}

// ShowLogin affiche le formulaire de connexion.
//
// Un utilisateur déjà connecté est renvoyé vers ses espaces : lui présenter
// un formulaire de connexion serait déroutant.
func (h *Handler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	if _, _, err := h.currentUser(r); err == nil {
		http.Redirect(w, r, "/spaces", http.StatusSeeOther)
		return
	}

	data := newPageData("Connexion", nil)
	data.Flash = takeFlash(w, r)
	data.Form = loginForm{}

	h.render(w, http.StatusOK, "login.html", data)
}

// Login traite l'envoi du formulaire de connexion.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	form := loginForm{Email: r.FormValue("email")}
	password := r.FormValue("password")

	result, err := h.api.Login(r.Context(), form.Email, password)
	if err != nil {
		// Un échec de connexion n'est pas une panne : on réaffiche le
		// formulaire avec le message, et non une page d'erreur.
		data := newPageData("Connexion", nil)
		data.Form = form
		applyAPIError(&data, err)

		if _, ok := api.AsError(err); !ok {
			log.Printf("connexion : %v", err)
		}

		// Le statut 401 est conservé dans la réponse : la page est bien un
		// refus d'authentification, pas un succès.
		h.render(w, statusForFormError(err), "login.html", data)
		return
	}

	// Le jeton est déposé dans le cookie, puis on redirige. La redirection
	// évite que le rechargement de la page rejoue le formulaire.
	session.Save(w, result.Token, result.ExpiresAt)
	http.Redirect(w, r, "/spaces", http.StatusSeeOther)
}

// ShowRegister affiche le formulaire d'inscription.
func (h *Handler) ShowRegister(w http.ResponseWriter, r *http.Request) {
	if _, _, err := h.currentUser(r); err == nil {
		http.Redirect(w, r, "/spaces", http.StatusSeeOther)
		return
	}

	data := newPageData("Créer un compte", nil)
	data.Form = registerForm{}

	h.render(w, http.StatusOK, "register.html", data)
}

// Register traite l'envoi du formulaire d'inscription.
//
// En cas de succès, l'utilisateur est connecté automatiquement : lui demander
// de ressaisir les identifiants qu'il vient de choisir n'aurait aucun intérêt.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	form := registerForm{
		Email: r.FormValue("email"),
		Name:  r.FormValue("name"),
	}
	password := r.FormValue("password")

	if _, err := h.api.Register(r.Context(), form.Email, password, form.Name); err != nil {
		data := newPageData("Créer un compte", nil)
		data.Form = form
		applyAPIError(&data, err)

		if _, ok := api.AsError(err); !ok {
			log.Printf("inscription : %v", err)
		}

		h.render(w, statusForFormError(err), "register.html", data)
		return
	}

	result, err := h.api.Login(r.Context(), form.Email, password)
	if err != nil {
		// Le compte est créé mais la connexion automatique a échoué. Plutôt
		// que d'afficher une erreur sur un compte qui existe bel et bien, on
		// envoie l'utilisateur vers la connexion avec un message explicite.
		log.Printf("connexion automatique après inscription : %v", err)
		setFlash(w, "Votre compte a été créé. Connectez-vous pour continuer.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session.Save(w, result.Token, result.ExpiresAt)
	setFlash(w, "Bienvenue ! Créez un premier espace pour ranger vos notes.")
	http.Redirect(w, r, "/spaces", http.StatusSeeOther)
}

// Logout efface la session et renvoie vers la page de connexion.
//
// La déconnexion passe par un POST et non par un lien : un lien de
// déconnexion peut être déclenché par le préchargement d'un navigateur ou par
// une image distante, ce qui déconnecterait l'utilisateur à son insu.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	session.Clear(w)
	setFlash(w, "Vous êtes déconnecté.")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Home oriente le visiteur selon qu'il a une session valide ou non.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if _, _, err := h.currentUser(r); err != nil {
		redirectToLogin(w, r)
		return
	}
	http.Redirect(w, r, "/spaces", http.StatusSeeOther)
}

// statusForFormError choisit le code de statut du réaffichage d'un formulaire.
//
// Conserver le statut de l'API (400, 401, 409) plutôt que de répondre 200
// garde la réponse honnête : un formulaire réaffiché après un refus n'est pas
// un succès. Les erreurs de transport donnent un 503, qui décrit bien la
// situation : le service dont dépend cette page est indisponible.
func statusForFormError(err error) int {
	if apiErr, ok := api.AsError(err); ok {
		return apiErr.StatusCode
	}
	return http.StatusServiceUnavailable
}
