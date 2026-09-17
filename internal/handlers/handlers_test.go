package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
	"github.com/KxroTM/test-technique-ynov-client/internal/render"
	"github.com/KxroTM/test-technique-ynov-client/web"
)

// Les tests de ce package montent l'application complète — routeur, handlers
// et gabarits réels — devant une fausse API. Ils vérifient donc ce qu'un
// navigateur recevrait vraiment : codes de statut, redirections, cookies et
// HTML produit.
//
// Aucun vrai serveur API ni aucune base de données n'est nécessaire.

// testApp regroupe l'application sous test et son client HTTP.
type testApp struct {
	server *httptest.Server
	client *http.Client
}

// newTestApp monte le client web devant la fausse API fournie.
func newTestApp(t *testing.T, apiHandler http.Handler) *testApp {
	t.Helper()

	apiServer := httptest.NewServer(apiHandler)
	t.Cleanup(apiServer.Close)

	// Les gabarits réels sont utilisés : un test qui passerait avec de faux
	// gabarits ne prouverait rien du HTML réellement produit.
	renderer, err := render.New(web.Files)
	if err != nil {
		t.Fatalf("compilation des gabarits : %v", err)
	}

	staticHandler, err := renderer.StaticHandler()
	if err != nil {
		t.Fatalf("accès aux fichiers statiques : %v", err)
	}

	handler := New(api.NewClient(apiServer.URL), renderer)

	webServer := httptest.NewServer(handler.Routes(staticHandler))
	t.Cleanup(webServer.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("création du bocal à cookies : %v", err)
	}

	return &testApp{
		server: webServer,
		client: &http.Client{
			Jar: jar,
			// Les redirections ne sont pas suivies : on veut observer le
			// statut 303 et sa destination, pas la page d'arrivée.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// get exécute une requête GET sur l'application.
func (a *testApp) get(t *testing.T, path string) (*http.Response, string) {
	t.Helper()

	response, err := a.client.Get(a.server.URL + path)
	if err != nil {
		t.Fatalf("GET %s : %v", path, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("lecture du corps de GET %s : %v", path, err)
	}

	return response, string(body)
}

// postForm exécute une soumission de formulaire sur l'application.
func (a *testApp) postForm(t *testing.T, path string, values url.Values) (*http.Response, string) {
	t.Helper()

	response, err := a.client.PostForm(a.server.URL+path, values)
	if err != nil {
		t.Fatalf("POST %s : %v", path, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("lecture du corps de POST %s : %v", path, err)
	}

	return response, string(body)
}

// hasSessionCookie indique si le bocal contient un cookie de session non vide.
func (a *testApp) hasSessionCookie(t *testing.T) bool {
	t.Helper()

	parsed, err := url.Parse(a.server.URL)
	if err != nil {
		t.Fatalf("URL du serveur de test invalide : %v", err)
	}

	for _, cookie := range a.client.Jar.Cookies(parsed) {
		if cookie.Name == "session_token" && cookie.Value != "" {
			return true
		}
	}

	return false
}

// ---------------------------------------------------------------------------
// Fausses API
// ---------------------------------------------------------------------------

// workingAPI simule une API qui répond correctement pour l'utilisateur Alice.
//
// Le jeton attendu est fixe : toute requête portant « Bearer jeton-valide »
// est acceptée, les autres reçoivent un 401. Cela suffit à exercer le
// comportement du client sans reproduire la logique du serveur.
func workingAPI() http.Handler {
	const validToken = "Bearer jeton-valide"

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var credentials struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		decodeJSON(r, &credentials)

		if credentials.Email != "alice@example.com" || credentials.Password != "password123" {
			writeJSON(w, http.StatusUnauthorized, `{"error":"email ou mot de passe incorrect"}`)
			return
		}

		writeJSON(w, http.StatusOK, `{
			"token": "jeton-valide",
			"expires_at": "2099-01-01T00:00:00Z",
			"user": {"id": 1, "email": "alice@example.com", "name": "Alice"}
		}`)
	})

	mux.HandleFunc("GET /api/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != validToken {
			writeJSON(w, http.StatusUnauthorized, `{"error":"authentification requise"}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"id": 1, "email": "alice@example.com", "name": "Alice"}`)
	})

	mux.HandleFunc("GET /api/spaces", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != validToken {
			writeJSON(w, http.StatusUnauthorized, `{"error":"authentification requise"}`)
			return
		}
		writeJSON(w, http.StatusOK, `[
			{"id": 1, "user_id": 1, "name": "Devoirs", "description": "Travaux", "note_count": 2},
			{"id": 2, "user_id": 1, "name": "Jobs", "description": "", "note_count": 1}
		]`)
	})

	// Espace appartenant à un autre utilisateur : l'API répond 404, jamais
	// 403, afin de ne pas révéler son existence.
	mux.HandleFunc("GET /api/spaces/99/notes", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, `{"error":"ressource introuvable"}`)
	})

	// Espace contenant une note dont le titre porte une charge XSS.
	mux.HandleFunc("GET /api/spaces/1/notes", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != validToken {
			writeJSON(w, http.StatusUnauthorized, `{"error":"authentification requise"}`)
			return
		}
		writeJSON(w, http.StatusOK, `{
			"space": {"id": 1, "user_id": 1, "name": "Devoirs", "description": "", "note_count": 1},
			"notes": [{
				"id": 10, "space_id": 1,
				"title": "<script>alert(1)</script>",
				"content": "Contenu",
				"status": "todo"
			}]
		}`)
	})

	mux.HandleFunc("POST /api/spaces", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != validToken {
			writeJSON(w, http.StatusUnauthorized, `{"error":"authentification requise"}`)
			return
		}

		var body struct {
			Name string `json:"name"`
		}
		decodeJSON(r, &body)

		if body.Name == "" {
			writeJSON(w, http.StatusBadRequest,
				`{"error":"données invalides","fields":{"name":"ce champ est obligatoire"}}`)
			return
		}

		writeJSON(w, http.StatusCreated,
			`{"id": 5, "user_id": 1, "name": "`+body.Name+`", "description": "", "note_count": 0}`)
	})

	return mux
}

// brokenAPI simule une API en défaut : elle répond 500 à tout.
func brokenAPI() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusInternalServerError, `{"error":"une erreur interne est survenue"}`)
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(body))
}

func decodeJSON(r *http.Request, target any) {
	// Les erreurs de décodage sont ignorées : la fausse API n'a pas à être
	// robuste, elle sert uniquement de partenaire aux tests.
	body, _ := io.ReadAll(r.Body)
	if len(body) > 0 {
		_ = json.Unmarshal(body, target)
	}
}

// ---------------------------------------------------------------------------
// Accès sans session
// ---------------------------------------------------------------------------

// Toutes les pages protégées doivent renvoyer vers la connexion lorsqu'aucune
// session n'est présente. C'est l'équivalent, côté navigateur, du 401 que
// renvoie le serveur.
func TestProtectedPagesRedirectWithoutSession(t *testing.T) {
	app := newTestApp(t, workingAPI())

	protectedPaths := []string{
		"/",
		"/spaces",
		"/spaces/new",
		"/spaces/1",
		"/spaces/1/edit",
		"/spaces/1/notes/new",
		"/notes/1/edit",
	}

	for _, path := range protectedPaths {
		t.Run(path, func(t *testing.T) {
			response, _ := app.get(t, path)

			if response.StatusCode != http.StatusSeeOther {
				t.Errorf("statut = %d, attendu %d", response.StatusCode, http.StatusSeeOther)
			}
			if location := response.Header.Get("Location"); location != "/login" {
				t.Errorf("Location = %q, attendu \"/login\"", location)
			}
		})
	}
}

// Les actions de modification sont aussi protégées : sans session, elles ne
// doivent pas être exécutées.
func TestProtectedActionsRedirectWithoutSession(t *testing.T) {
	app := newTestApp(t, workingAPI())

	protectedActions := []string{
		"/spaces/new",
		"/spaces/1/edit",
		"/spaces/1/delete",
		"/spaces/1/notes/new",
		"/notes/1/edit",
		"/notes/1/delete",
	}

	for _, path := range protectedActions {
		t.Run(path, func(t *testing.T) {
			response, _ := app.postForm(t, path, url.Values{"name": {"X"}})

			if response.StatusCode != http.StatusSeeOther {
				t.Errorf("statut = %d, attendu %d", response.StatusCode, http.StatusSeeOther)
			}
			if location := response.Header.Get("Location"); location != "/login" {
				t.Errorf("Location = %q, attendu \"/login\"", location)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Connexion
// ---------------------------------------------------------------------------

func TestLoginPageIsPublic(t *testing.T) {
	app := newTestApp(t, workingAPI())

	response, body := app.get(t, "/login")

	if response.StatusCode != http.StatusOK {
		t.Errorf("statut = %d, attendu 200", response.StatusCode)
	}
	if !strings.Contains(body, "Connexion") {
		t.Error("la page ne contient pas le titre attendu")
	}
}

func TestLoginSuccessSetsSessionAndRedirects(t *testing.T) {
	app := newTestApp(t, workingAPI())

	response, _ := app.postForm(t, "/login", url.Values{
		"email":    {"alice@example.com"},
		"password": {"password123"},
	})

	if response.StatusCode != http.StatusSeeOther {
		t.Errorf("statut = %d, attendu %d", response.StatusCode, http.StatusSeeOther)
	}
	if location := response.Header.Get("Location"); location != "/spaces" {
		t.Errorf("Location = %q, attendu \"/spaces\"", location)
	}
	if !app.hasSessionCookie(t) {
		t.Error("aucun cookie de session déposé après une connexion réussie")
	}
}

// Un échec de connexion réaffiche le formulaire, et non une page d'erreur.
// L'email saisi est conservé, le mot de passe jamais.
func TestLoginFailureRedisplaysFormWithMessage(t *testing.T) {
	app := newTestApp(t, workingAPI())

	response, body := app.postForm(t, "/login", url.Values{
		"email":    {"alice@example.com"},
		"password": {"mauvais-mot-de-passe"},
	})

	// Le statut de l'API est conservé : un formulaire réaffiché après un
	// refus n'est pas un succès.
	if response.StatusCode != http.StatusUnauthorized {
		t.Errorf("statut = %d, attendu %d", response.StatusCode, http.StatusUnauthorized)
	}
	if !strings.Contains(body, "email ou mot de passe incorrect") {
		t.Error("le message d'erreur de l'API n'est pas affiché")
	}
	if !strings.Contains(body, `value="alice@example.com"`) {
		t.Error("l'email saisi n'est pas réaffiché")
	}
	if strings.Contains(body, "mauvais-mot-de-passe") {
		t.Error("le mot de passe saisi est réaffiché dans le HTML")
	}
	if app.hasSessionCookie(t) {
		t.Error("un cookie de session a été déposé malgré l'échec")
	}
}

// ---------------------------------------------------------------------------
// Pages authentifiées
// ---------------------------------------------------------------------------

// login connecte l'application de test et vérifie que la session est établie.
func (a *testApp) login(t *testing.T) {
	t.Helper()

	response, _ := a.postForm(t, "/login", url.Values{
		"email":    {"alice@example.com"},
		"password": {"password123"},
	})

	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("connexion en échec : statut %d", response.StatusCode)
	}
	if !a.hasSessionCookie(t) {
		t.Fatal("connexion en échec : aucun cookie de session")
	}
}

func TestSpacesPageListsSpaces(t *testing.T) {
	app := newTestApp(t, workingAPI())
	app.login(t)

	response, body := app.get(t, "/spaces")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("statut = %d, attendu 200", response.StatusCode)
	}

	for _, expected := range []string{"Devoirs", "Jobs", "Alice"} {
		if !strings.Contains(body, expected) {
			t.Errorf("la page ne contient pas %q", expected)
		}
	}

	// Le pluriel est calculé dans le gabarit : deux notes pour le premier
	// espace, une seule pour le second.
	if !strings.Contains(body, "2 notes") {
		t.Error("le nombre de notes au pluriel n'est pas affiché")
	}
	if !strings.Contains(body, "1 note") {
		t.Error("le nombre de notes au singulier n'est pas affiché")
	}
}

// Un espace appartenant à un autre utilisateur donne un 404 de l'API, que le
// client traduit en page « introuvable ». Le client ne réimplémente aucun
// contrôle d'accès : il se contente de présenter la réponse du serveur.
func TestSpaceOfAnotherUserRendersNotFound(t *testing.T) {
	app := newTestApp(t, workingAPI())
	app.login(t)

	response, body := app.get(t, "/spaces/99")

	if response.StatusCode != http.StatusNotFound {
		t.Errorf("statut = %d, attendu 404", response.StatusCode)
	}
	if !strings.Contains(body, "Cette page n") {
		t.Error("la page « introuvable » n'est pas affichée")
	}
}

// Un identifiant non numérique ne doit pas provoquer d'erreur serveur.
func TestInvalidIDRendersNotFound(t *testing.T) {
	app := newTestApp(t, workingAPI())
	app.login(t)

	for _, path := range []string{"/spaces/abc", "/spaces/-1", "/notes/abc/edit"} {
		t.Run(path, func(t *testing.T) {
			response, _ := app.get(t, path)
			if response.StatusCode != http.StatusNotFound {
				t.Errorf("statut = %d, attendu 404", response.StatusCode)
			}
		})
	}
}

func TestUnknownRouteRendersNotFoundPage(t *testing.T) {
	app := newTestApp(t, workingAPI())

	response, body := app.get(t, "/adresse-inconnue")

	if response.StatusCode != http.StatusNotFound {
		t.Errorf("statut = %d, attendu 404", response.StatusCode)
	}
	// La page d'erreur maison est rendue, et non le « 404 page not found »
	// brut de la bibliothèque standard.
	if strings.Contains(body, "404 page not found") {
		t.Error("le 404 brut de net/http est renvoyé au lieu de la page d'erreur")
	}
	if !strings.Contains(body, "error-code") {
		t.Error("la page d'erreur mise en forme n'est pas rendue")
	}
}

// ---------------------------------------------------------------------------
// Échappement
// ---------------------------------------------------------------------------

// html/template échappe selon le contexte. Une note dont le titre contient
// une balise doit s'afficher comme du texte, sans qu'aucun échappement
// n'ait à être écrit à la main dans les gabarits.
func TestNoteTitleIsEscaped(t *testing.T) {
	app := newTestApp(t, workingAPI())
	app.login(t)

	_, body := app.get(t, "/spaces/1")

	// La balise ne doit jamais apparaître telle quelle : elle serait alors
	// exécutée par le navigateur.
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Error("la charge XSS est rendue sans échappement")
	}

	// Dans le corps de la page, le contexte est du texte HTML : les
	// chevrons sont remplacés par des entités.
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Error("le titre n'est pas échappé en contexte HTML")
	}

	// Dans l'attribut onsubmit, le contexte est une chaîne JavaScript :
	// html/template y applique un échappement différent, sous forme de
	// séquences \u. C'est la démonstration de l'échappement contextuel : le
	// même titre est encodé de deux façons selon l'endroit où il apparaît.
	// Le marqueur « u003cscript » est cherché sans son antislash : il suffit
	// à prouver que le chevron a été encodé en séquence unicode, forme
	// propre au contexte JavaScript et absente du contexte HTML.
	if !strings.Contains(body, "u003cscript") {
		t.Error("le titre n'est pas échappé en contexte JavaScript")
	}
}

// ---------------------------------------------------------------------------
// Validation et messages de confirmation
// ---------------------------------------------------------------------------

// Une erreur de validation de l'API doit réafficher le formulaire avec le
// message placé sous le champ concerné.
func TestSpaceCreationValidationErrorRedisplaysForm(t *testing.T) {
	app := newTestApp(t, workingAPI())
	app.login(t)

	response, body := app.postForm(t, "/spaces/new", url.Values{
		"name":        {""},
		"description": {"Une description"},
	})

	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("statut = %d, attendu 400", response.StatusCode)
	}
	if !strings.Contains(body, "ce champ est obligatoire") {
		t.Error("le message d'erreur du champ n'est pas affiché")
	}
	if !strings.Contains(body, "field-error") {
		t.Error("le message n'est pas rendu comme une erreur de champ")
	}
	// La description saisie doit être conservée pour ne pas obliger
	// l'utilisateur à tout retaper.
	if !strings.Contains(body, "Une description") {
		t.Error("la description saisie n'est pas réaffichée")
	}
}

// Après une création, l'application redirige et affiche un message de
// confirmation qui ne doit apparaître qu'une seule fois.
func TestFlashMessageIsShownOnceAfterRedirect(t *testing.T) {
	app := newTestApp(t, workingAPI())
	app.login(t)

	response, _ := app.postForm(t, "/spaces/new", url.Values{
		"name":        {"Projets"},
		"description": {""},
	})

	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("statut = %d, attendu %d", response.StatusCode, http.StatusSeeOther)
	}

	// Première consultation : le message est présent.
	_, firstBody := app.get(t, "/spaces")
	if !strings.Contains(firstBody, "a été créé") {
		t.Error("le message de confirmation n'est pas affiché après la redirection")
	}

	// Seconde consultation : il a disparu. Un message de confirmation qui
	// survit au rechargement serait trompeur.
	_, secondBody := app.get(t, "/spaces")
	if strings.Contains(secondBody, "a été créé") {
		t.Error("le message de confirmation est encore affiché au rechargement")
	}
}

// ---------------------------------------------------------------------------
// Résilience
// ---------------------------------------------------------------------------

// Une API en défaut ne doit pas déconnecter l'utilisateur : son jeton est
// valide, c'est le service distant qui est en cause. Le déconnecter
// l'obligerait à se reconnecter sans raison.
func TestAPIFailureDoesNotDestroySession(t *testing.T) {
	app := newTestApp(t, workingAPI())
	app.login(t)

	// L'application est reconstruite devant une API en défaut, en
	// conservant le bocal à cookies : la session reste donc établie.
	brokenAPIServer := httptest.NewServer(brokenAPI())
	t.Cleanup(brokenAPIServer.Close)

	renderer, err := render.New(web.Files)
	if err != nil {
		t.Fatalf("compilation des gabarits : %v", err)
	}
	staticHandler, err := renderer.StaticHandler()
	if err != nil {
		t.Fatalf("accès aux fichiers statiques : %v", err)
	}

	handler := New(api.NewClient(brokenAPIServer.URL), renderer)
	brokenApp := httptest.NewServer(handler.Routes(staticHandler))
	t.Cleanup(brokenApp.Close)

	// Les cookies suivent le domaine : on rejoue la requête avec le même
	// client, dont le bocal porte déjà la session.
	previousServer := app.server
	app.server = brokenApp
	t.Cleanup(func() { app.server = previousServer })

	response, body := app.get(t, "/spaces")

	if response.StatusCode != http.StatusInternalServerError {
		t.Errorf("statut = %d, attendu 500", response.StatusCode)
	}
	if strings.Contains(body, "Connexion") {
		t.Error("l'utilisateur a été renvoyé vers la connexion malgré une session valide")
	}
	if !app.hasSessionCookie(t) {
		t.Error("le cookie de session a été effacé alors que l'API seule est en cause")
	}
}

// Un cookie de session refusé par l'API doit en revanche provoquer une
// déconnexion propre, et non une page d'erreur.
func TestRejectedTokenClearsSessionAndRedirects(t *testing.T) {
	app := newTestApp(t, workingAPI())

	// Cookie forgé : la fausse API ne reconnaît que « jeton-valide ».
	parsed, err := url.Parse(app.server.URL)
	if err != nil {
		t.Fatalf("URL du serveur de test invalide : %v", err)
	}
	app.client.Jar.SetCookies(parsed, []*http.Cookie{
		{Name: "session_token", Value: "jeton-invalide", Path: "/"},
	})

	response, _ := app.get(t, "/spaces")

	if response.StatusCode != http.StatusSeeOther {
		t.Errorf("statut = %d, attendu %d", response.StatusCode, http.StatusSeeOther)
	}
	if location := response.Header.Get("Location"); location != "/login" {
		t.Errorf("Location = %q, attendu \"/login\"", location)
	}
	if app.hasSessionCookie(t) {
		t.Error("le cookie de session n'a pas été effacé après le refus du jeton")
	}
}

// ---------------------------------------------------------------------------
// Déconnexion
// ---------------------------------------------------------------------------

func TestLogoutClearsSession(t *testing.T) {
	app := newTestApp(t, workingAPI())
	app.login(t)

	response, _ := app.postForm(t, "/logout", nil)

	if response.StatusCode != http.StatusSeeOther {
		t.Errorf("statut = %d, attendu %d", response.StatusCode, http.StatusSeeOther)
	}
	if location := response.Header.Get("Location"); location != "/login" {
		t.Errorf("Location = %q, attendu \"/login\"", location)
	}
	if app.hasSessionCookie(t) {
		t.Error("le cookie de session subsiste après la déconnexion")
	}
}

// La déconnexion n'est accessible qu'en POST : un lien GET pourrait être
// déclenché par le préchargement d'un navigateur ou par une image distante,
// et déconnecterait l'utilisateur à son insu.
func TestLogoutRejectsGet(t *testing.T) {
	app := newTestApp(t, workingAPI())
	app.login(t)

	response, _ := app.get(t, "/logout")

	if response.StatusCode == http.StatusSeeOther {
		t.Error("la déconnexion a été acceptée en GET")
	}
	if !app.hasSessionCookie(t) {
		t.Error("la session a été détruite par une simple requête GET")
	}
}

// ---------------------------------------------------------------------------
// Fichiers statiques
// ---------------------------------------------------------------------------

func TestStyleSheetIsServed(t *testing.T) {
	app := newTestApp(t, workingAPI())

	response, body := app.get(t, "/static/style.css")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("statut = %d, attendu 200", response.StatusCode)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/css") {
		t.Errorf("Content-Type = %q, attendu text/css", contentType)
	}
	if len(body) == 0 {
		t.Error("la feuille de style est vide")
	}
}
