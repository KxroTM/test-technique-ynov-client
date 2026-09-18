// Package render assemble et exécute les gabarits HTML

package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
)

// Renderer conserve les gabarits compilés
type Renderer struct {
	pages map[string]*template.Template

	files fs.FS
}

// New compile l'ensemble des gabarits présents dans le système de fichiers
func New(files fs.FS) (*Renderer, error) {
	pagePaths, err := fs.Glob(files, "templates/pages/*.html")
	if err != nil {
		return nil, fmt.Errorf("recherche des pages : %w", err)
	}
	if len(pagePaths) == 0 {
		return nil, fmt.Errorf("aucune page trouvée dans templates/pages")
	}

	renderer := &Renderer{
		pages: make(map[string]*template.Template, len(pagePaths)),
		files: files,
	}

	for _, pagePath := range pagePaths {
		name := path.Base(pagePath)

		patterns := []string{
			"templates/layout.html",
			"templates/partials/*.html",
			pagePath,
		}

		compiled, err := template.New(name).Funcs(templateFuncs()).ParseFS(files, patterns...)
		if err != nil {
			return nil, fmt.Errorf("compilation de la page %s : %w", name, err)
		}

		renderer.pages[name] = compiled
	}

	return renderer, nil
}

// StaticHandler retourne un handler servant les fichiers statiques
func (r *Renderer) StaticHandler() (http.Handler, error) {
	staticFS, err := fs.Sub(r.files, "static")
	if err != nil {
		return nil, fmt.Errorf("accès aux fichiers statiques : %w", err)
	}
	return http.FileServer(http.FS(staticFS)), nil
}

// Page rend une page avec le code de statut fourni
func (r *Renderer) Page(w http.ResponseWriter, statusCode int, name string, data any) error {
	compiled, ok := r.pages[name]
	if !ok {
		return fmt.Errorf("page inconnue : %s", name)
	}

	var buffer bytes.Buffer
	if err := compiled.ExecuteTemplate(&buffer, "layout", data); err != nil {
		return fmt.Errorf("rendu de la page %s : %w", name, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)

	if _, err := buffer.WriteTo(w); err != nil {
		return fmt.Errorf("écriture de la réponse : %w", err)
	}

	return nil
}
