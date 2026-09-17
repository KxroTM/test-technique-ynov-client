package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Les tests de ce package utilisent un faux serveur API monté avec httptest.
// Aucun vrai serveur n'est nécessaire : on vérifie ce que le client émet
// (méthode, chemin, en-tête d'authentification, corps) et comment il
// interprète ce qu'il reçoit.

// recordedRequest conserve ce que le faux serveur a reçu, pour pouvoir
// l'inspecter après l'appel.
type recordedRequest struct {
	Method        string
	Path          string
	Authorization string
	Body          string
}

// newTestClient monte un faux serveur répondant avec le statut et le corps
// fournis, et retourne un client pointant vers lui.
func newTestClient(t *testing.T, statusCode int, responseBody string) (*Client, *recordedRequest) {
	t.Helper()

	recorded := &recordedRequest{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		recorded.Method = r.Method
		recorded.Path = r.URL.Path
		recorded.Authorization = r.Header.Get("Authorization")
		recorded.Body = string(body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if responseBody != "" {
			_, _ = w.Write([]byte(responseBody))
		}
	}))

	t.Cleanup(server.Close)

	return NewClient(server.URL), recorded
}

// Le jeton doit être transmis dans l'en-tête Authorization au format Bearer :
// c'est ce que le middleware du serveur attend.
func TestClientSendsBearerToken(t *testing.T) {
	client, recorded := newTestClient(t, http.StatusOK, `[]`)

	if _, err := client.ListSpaces(context.Background(), "mon-jeton"); err != nil {
		t.Fatalf("ListSpaces a échoué : %v", err)
	}

	if want := "Bearer mon-jeton"; recorded.Authorization != want {
		t.Errorf("Authorization = %q, attendu %q", recorded.Authorization, want)
	}
}

// Les routes publiques ne doivent pas envoyer d'en-tête d'authentification.
func TestClientOmitsTokenOnPublicRoutes(t *testing.T) {
	client, recorded := newTestClient(t, http.StatusOK, `{"token":"t","user":{"id":1}}`)

	if _, err := client.Login(context.Background(), "alice@example.com", "password123"); err != nil {
		t.Fatalf("Login a échoué : %v", err)
	}

	if recorded.Authorization != "" {
		t.Errorf("Authorization = %q, aucun en-tête n'est attendu sur /auth/login", recorded.Authorization)
	}
}

// Vérifie que chaque méthode du client émet bien le verbe et le chemin
// attendus par l'API.
//
// C'est le test qui verrouille la traduction des actions du client : le
// navigateur envoie un POST (un formulaire HTML ne sait rien faire d'autre),
// et c'est ici que l'action devient un PUT ou un DELETE côté API.
func TestClientUsesExpectedMethodAndPath(t *testing.T) {
	testCases := []struct {
		name       string
		call       func(*Client) error
		wantMethod string
		wantPath   string
	}{
		{
			name:       "liste des espaces",
			call:       func(c *Client) error { _, err := c.ListSpaces(context.Background(), "t"); return err },
			wantMethod: http.MethodGet,
			wantPath:   "/api/spaces",
		},
		{
			name: "création d'un espace",
			call: func(c *Client) error {
				_, err := c.CreateSpace(context.Background(), "t", "Devoirs", "")
				return err
			},
			wantMethod: http.MethodPost,
			wantPath:   "/api/spaces",
		},
		{
			name: "modification d'un espace",
			call: func(c *Client) error {
				_, err := c.UpdateSpace(context.Background(), "t", 7, "Devoirs", "")
				return err
			},
			wantMethod: http.MethodPut,
			wantPath:   "/api/spaces/7",
		},
		{
			name:       "suppression d'un espace",
			call:       func(c *Client) error { return c.DeleteSpace(context.Background(), "t", 7) },
			wantMethod: http.MethodDelete,
			wantPath:   "/api/spaces/7",
		},
		{
			name: "notes d'un espace",
			call: func(c *Client) error {
				_, err := c.ListSpaceNotes(context.Background(), "t", 7)
				return err
			},
			wantMethod: http.MethodGet,
			wantPath:   "/api/spaces/7/notes",
		},
		{
			name: "création d'une note",
			call: func(c *Client) error {
				_, err := c.CreateNote(context.Background(), "t", 7, "Titre", "", StatusTodo)
				return err
			},
			wantMethod: http.MethodPost,
			wantPath:   "/api/spaces/7/notes",
		},
		{
			name: "modification d'une note",
			call: func(c *Client) error {
				_, err := c.UpdateNote(context.Background(), "t", 3, "Titre", "", StatusDone)
				return err
			},
			wantMethod: http.MethodPut,
			wantPath:   "/api/notes/3",
		},
		{
			name:       "suppression d'une note",
			call:       func(c *Client) error { return c.DeleteNote(context.Background(), "t", 3) },
			wantMethod: http.MethodDelete,
			wantPath:   "/api/notes/3",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Le corps « null » se décode indifféremment dans une structure
			// ou dans un slice. Ce test ne s'intéresse qu'à la requête
			// émise, pas à la réponse : un corps neutre convient donc pour
			// tous les appels du tableau.
			client, recorded := newTestClient(t, http.StatusOK, `null`)

			if err := testCase.call(client); err != nil {
				t.Fatalf("appel en échec : %v", err)
			}

			if recorded.Method != testCase.wantMethod {
				t.Errorf("méthode = %s, attendu %s", recorded.Method, testCase.wantMethod)
			}
			if recorded.Path != testCase.wantPath {
				t.Errorf("chemin = %s, attendu %s", recorded.Path, testCase.wantPath)
			}
		})
	}
}

// Une suppression répond 204 sans corps : le client ne doit pas tenter de
// décoder une réponse vide, ce qui produirait une erreur trompeuse.
func TestClientAcceptsNoContentResponse(t *testing.T) {
	client, _ := newTestClient(t, http.StatusNoContent, "")

	if err := client.DeleteNote(context.Background(), "t", 1); err != nil {
		t.Errorf("DeleteNote a échoué sur une réponse 204 : %v", err)
	}
}

// Les erreurs de validation de l'API doivent être restituées champ par champ :
// c'est ce qui permet au client d'afficher le message sous le bon champ.
func TestClientParsesValidationError(t *testing.T) {
	const body = `{"error":"données invalides","fields":{"title":"ce champ est obligatoire"}}`

	client, _ := newTestClient(t, http.StatusBadRequest, body)

	_, err := client.CreateNote(context.Background(), "t", 1, "", "", StatusTodo)
	if err == nil {
		t.Fatal("aucune erreur retournée alors que l'API a répondu 400")
	}

	apiErr, ok := AsError(err)
	if !ok {
		t.Fatalf("erreur de type %T, une *Error est attendue", err)
	}

	if !apiErr.IsValidation() {
		t.Error("IsValidation() = false, attendu true pour un statut 400")
	}
	if want := "ce champ est obligatoire"; apiErr.FieldError("title") != want {
		t.Errorf("FieldError(title) = %q, attendu %q", apiErr.FieldError("title"), want)
	}
}

func TestClientClassifiesStatusCodes(t *testing.T) {
	testCases := []struct {
		statusCode       int
		wantUnauthorized bool
		wantNotFound     bool
		wantValidation   bool
	}{
		{http.StatusUnauthorized, true, false, false},
		{http.StatusNotFound, false, true, false},
		{http.StatusBadRequest, false, false, true},
		{http.StatusConflict, false, false, true},
		{http.StatusInternalServerError, false, false, false},
	}

	for _, testCase := range testCases {
		client, _ := newTestClient(t, testCase.statusCode, `{"error":"message"}`)

		_, err := client.GetSpace(context.Background(), "t", 1)
		apiErr, ok := AsError(err)
		if !ok {
			t.Fatalf("statut %d : erreur de type %T, une *Error est attendue", testCase.statusCode, err)
		}

		if apiErr.IsUnauthorized() != testCase.wantUnauthorized {
			t.Errorf("statut %d : IsUnauthorized() = %v, attendu %v",
				testCase.statusCode, apiErr.IsUnauthorized(), testCase.wantUnauthorized)
		}
		if apiErr.IsNotFound() != testCase.wantNotFound {
			t.Errorf("statut %d : IsNotFound() = %v, attendu %v",
				testCase.statusCode, apiErr.IsNotFound(), testCase.wantNotFound)
		}
		if apiErr.IsValidation() != testCase.wantValidation {
			t.Errorf("statut %d : IsValidation() = %v, attendu %v",
				testCase.statusCode, apiErr.IsValidation(), testCase.wantValidation)
		}
	}
}

// Si le corps d'erreur ne suit pas le format de l'API — proxy renvoyant du
// HTML, serveur en défaut — le client doit quand même produire une erreur
// exploitable plutôt que d'échouer au décodage.
func TestClientHandlesUnexpectedErrorBody(t *testing.T) {
	client, _ := newTestClient(t, http.StatusBadGateway, "<html>502 Bad Gateway</html>")

	_, err := client.GetSpace(context.Background(), "t", 1)

	apiErr, ok := AsError(err)
	if !ok {
		t.Fatalf("erreur de type %T, une *Error est attendue", err)
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Errorf("StatusCode = %d, attendu %d", apiErr.StatusCode, http.StatusBadGateway)
	}
	if apiErr.Message == "" {
		t.Error("le message d'erreur est vide")
	}
}

// Une API injoignable ne doit pas produire une *Error : il n'y a pas de code
// de statut. Le client doit pouvoir distinguer ce cas d'un refus applicatif,
// car il ne faut pas déconnecter l'utilisateur pour une panne réseau.
func TestClientReportsTransportErrorWithoutStatus(t *testing.T) {
	// Adresse volontairement injoignable.
	client := NewClient("http://127.0.0.1:1")

	_, err := client.ListSpaces(context.Background(), "t")
	if err == nil {
		t.Fatal("aucune erreur retournée alors que l'API est injoignable")
	}

	if _, ok := AsError(err); ok {
		t.Error("une *Error a été produite pour une erreur de transport")
	}
	if IsUnauthorized(err) {
		t.Error("IsUnauthorized() = true pour une erreur de transport")
	}
}

// Le corps envoyé doit contenir les champs attendus par l'API, avec les noms
// JSON qu'elle utilise.
func TestClientSendsExpectedJSONBody(t *testing.T) {
	client, recorded := newTestClient(t, http.StatusCreated, `{}`)

	_, err := client.CreateNote(context.Background(), "t", 1, "Mon titre", "Mon contenu", StatusInProgress)
	if err != nil {
		t.Fatalf("CreateNote a échoué : %v", err)
	}

	var sent map[string]any
	if err := json.Unmarshal([]byte(recorded.Body), &sent); err != nil {
		t.Fatalf("le corps envoyé n'est pas du JSON valide : %v", err)
	}

	expected := map[string]any{
		"title":   "Mon titre",
		"content": "Mon contenu",
		"status":  "in_progress",
	}

	for field, want := range expected {
		if sent[field] != want {
			t.Errorf("champ %q = %v, attendu %v", field, sent[field], want)
		}
	}
}

func TestNoteStatusLabel(t *testing.T) {
	testCases := map[NoteStatus]string{
		StatusTodo:       "Non fait",
		StatusInProgress: "En cours",
		StatusDone:       "Terminé",
	}

	for status, want := range testCases {
		if got := status.Label(); got != want {
			t.Errorf("NoteStatus(%q).Label() = %q, attendu %q", status, got, want)
		}
	}
}
