package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// maxResponseSize borne la taille des réponses lues depuis l'API.
//
// Sans cette limite, une réponse anormalement volumineuse — serveur en
// défaut, mauvaise adresse pointant vers autre chose — pourrait saturer la
// mémoire du client.
const maxResponseSize = 5 << 20 // 5 Mio

// Client dialogue avec le serveur API.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient construit le client.
//
// Le http.Client est créé ici avec un délai d'expiration explicite. Le client
// par défaut de la bibliothèque standard n'en a aucun : une API qui ne répond
// pas bloquerait la requête de l'utilisateur indéfiniment.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ---------------------------------------------------------------------------
// Authentification
// ---------------------------------------------------------------------------

// Register crée un compte utilisateur.
func (c *Client) Register(ctx context.Context, email, password, name string) (*User, error) {
	body := map[string]string{"email": email, "password": password, "name": name}

	var user User
	if err := c.do(ctx, http.MethodPost, "/api/auth/register", "", body, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// Login authentifie un utilisateur et retourne son jeton.
func (c *Client) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	body := map[string]string{"email": email, "password": password}

	var result LoginResult
	if err := c.do(ctx, http.MethodPost, "/api/auth/login", "", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Me retourne le profil de l'utilisateur authentifié.
func (c *Client) Me(ctx context.Context, token string) (*User, error) {
	var user User
	if err := c.do(ctx, http.MethodGet, "/api/me", token, nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ---------------------------------------------------------------------------
// Espaces
// ---------------------------------------------------------------------------

// ListSpaces retourne les espaces de l'utilisateur.
func (c *Client) ListSpaces(ctx context.Context, token string) ([]Space, error) {
	var spaces []Space
	if err := c.do(ctx, http.MethodGet, "/api/spaces", token, nil, &spaces); err != nil {
		return nil, err
	}
	return spaces, nil
}

// GetSpace retourne un espace de l'utilisateur.
func (c *Client) GetSpace(ctx context.Context, token string, spaceID int64) (*Space, error) {
	var space Space
	path := fmt.Sprintf("/api/spaces/%d", spaceID)
	if err := c.do(ctx, http.MethodGet, path, token, nil, &space); err != nil {
		return nil, err
	}
	return &space, nil
}

// CreateSpace crée un espace.
func (c *Client) CreateSpace(ctx context.Context, token, name, description string) (*Space, error) {
	body := map[string]string{"name": name, "description": description}

	var space Space
	if err := c.do(ctx, http.MethodPost, "/api/spaces", token, body, &space); err != nil {
		return nil, err
	}
	return &space, nil
}

// UpdateSpace modifie un espace.
func (c *Client) UpdateSpace(ctx context.Context, token string, spaceID int64, name, description string) (*Space, error) {
	body := map[string]string{"name": name, "description": description}

	var space Space
	path := fmt.Sprintf("/api/spaces/%d", spaceID)
	if err := c.do(ctx, http.MethodPut, path, token, body, &space); err != nil {
		return nil, err
	}
	return &space, nil
}

// DeleteSpace supprime un espace et ses notes.
func (c *Client) DeleteSpace(ctx context.Context, token string, spaceID int64) error {
	path := fmt.Sprintf("/api/spaces/%d", spaceID)
	return c.do(ctx, http.MethodDelete, path, token, nil, nil)
}

// ---------------------------------------------------------------------------
// Notes
// ---------------------------------------------------------------------------

// ListSpaceNotes retourne un espace accompagné de ses notes.
//
// L'API renvoie les deux dans une seule réponse : la page d'un espace a besoin
// de son nom pour son titre et de ses notes pour sa liste.
func (c *Client) ListSpaceNotes(ctx context.Context, token string, spaceID int64) (*SpaceWithNotes, error) {
	var result SpaceWithNotes
	path := fmt.Sprintf("/api/spaces/%d/notes", spaceID)
	if err := c.do(ctx, http.MethodGet, path, token, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateNote ajoute une note dans un espace.
func (c *Client) CreateNote(ctx context.Context, token string, spaceID int64, title, content string, status NoteStatus) (*Note, error) {
	body := map[string]any{"title": title, "content": content, "status": status}

	var note Note
	path := fmt.Sprintf("/api/spaces/%d/notes", spaceID)
	if err := c.do(ctx, http.MethodPost, path, token, body, &note); err != nil {
		return nil, err
	}
	return &note, nil
}

// GetNote retourne une note de l'utilisateur.
func (c *Client) GetNote(ctx context.Context, token string, noteID int64) (*Note, error) {
	var note Note
	path := fmt.Sprintf("/api/notes/%d", noteID)
	if err := c.do(ctx, http.MethodGet, path, token, nil, &note); err != nil {
		return nil, err
	}
	return &note, nil
}

// UpdateNote modifie une note.
func (c *Client) UpdateNote(ctx context.Context, token string, noteID int64, title, content string, status NoteStatus) (*Note, error) {
	body := map[string]any{"title": title, "content": content, "status": status}

	var note Note
	path := fmt.Sprintf("/api/notes/%d", noteID)
	if err := c.do(ctx, http.MethodPut, path, token, body, &note); err != nil {
		return nil, err
	}
	return &note, nil
}

// DeleteNote supprime une note.
func (c *Client) DeleteNote(ctx context.Context, token string, noteID int64) error {
	path := fmt.Sprintf("/api/notes/%d", noteID)
	return c.do(ctx, http.MethodDelete, path, token, nil, nil)
}

// ---------------------------------------------------------------------------
// Mécanique commune
// ---------------------------------------------------------------------------

// do exécute une requête vers l'API et décode la réponse.
//
// Toutes les méthodes publiques passent par ici. Centraliser la construction
// de la requête, l'ajout du jeton, le traitement des codes d'erreur et le
// décodage évite de répéter une quinzaine de fois la même séquence, et
// garantit que chaque appel se comporte de la même manière.
//
// requestBody et responseTarget peuvent être nil : une suppression n'envoie
// rien et ne retourne rien.
func (c *Client) do(ctx context.Context, method, path, token string, requestBody, responseTarget any) error {
	var bodyReader io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encodage de la requête : %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("construction de la requête : %w", err)
	}

	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := c.http.Do(request)
	if err != nil {
		// Erreur de transport : API arrêtée, adresse injoignable, délai
		// dépassé. Ce n'est pas une erreur applicative, elle ne porte donc
		// pas de code de statut.
		return fmt.Errorf("serveur injoignable : %w", err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize))
	if err != nil {
		return fmt.Errorf("lecture de la réponse : %w", err)
	}

	if response.StatusCode >= 400 {
		return parseErrorResponse(response.StatusCode, payload)
	}

	// 204 No Content : rien à décoder, comme pour les suppressions.
	if responseTarget == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}

	if err := json.Unmarshal(payload, responseTarget); err != nil {
		return fmt.Errorf("décodage de la réponse : %w", err)
	}

	return nil
}

// parseErrorResponse construit une *Error à partir d'une réponse en échec.
//
// Le corps est censé suivre le format d'erreur de l'API. S'il ne le suit pas
// — proxy renvoyant du HTML, serveur en défaut — on retombe sur un message
// générique construit depuis le code de statut, plutôt que d'échouer au
// décodage et de masquer l'erreur d'origine.
func parseErrorResponse(statusCode int, payload []byte) error {
	var decoded struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}

	if err := json.Unmarshal(payload, &decoded); err != nil || decoded.Error == "" {
		return &Error{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("réponse inattendue du serveur (code %d)", statusCode),
		}
	}

	return &Error{
		StatusCode: statusCode,
		Message:    decoded.Error,
		Fields:     decoded.Fields,
	}
}
