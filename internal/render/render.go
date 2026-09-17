// Package render assemble et exécute les gabarits HTML.
//
// Il reçoit un système de fichiers en paramètre plutôt que d'embarquer les
// gabarits lui-même : il reste ainsi indépendant de l'endroit où ils sont
// rangés, et peut être testé avec un fstest.MapFS en mémoire.
package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
)

// Renderer conserve les gabarits compilés.
type Renderer struct {
	// pages associe le nom d'une page à son gabarit complet, layout inclus.
	//
	// Les gabarits sont compilés une seule fois au démarrage, et non à
	// chaque requête : une erreur de syntaxe fait donc échouer le démarrage
	// du serveur plutôt que d'apparaître au hasard d'une navigation. C'est
	// aussi nettement plus rapide à l'exécution.
	pages map[string]*template.Template

	files fs.FS
}

// New compile l'ensemble des gabarits présents dans le système de fichiers.
//
// Chaque page est compilée séparément, avec le layout et les partials. Un
// unique gabarit global ne fonctionnerait pas : toutes les pages définissent
// un bloc « content » portant le même nom, et la dernière compilée écraserait
// les précédentes.
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

// StaticHandler retourne un handler servant les fichiers statiques.
func (r *Renderer) StaticHandler() (http.Handler, error) {
	staticFS, err := fs.Sub(r.files, "static")
	if err != nil {
		return nil, fmt.Errorf("accès aux fichiers statiques : %w", err)
	}
	return http.FileServer(http.FS(staticFS)), nil
}

// Page rend une page avec le code de statut fourni.
//
// Le rendu est d'abord effectué dans un tampon mémoire, puis recopié vers la
// réponse. Ce détour est important : si le gabarit échoue à mi-parcours
// (champ inexistant, pointeur nil), une écriture directe aurait déjà envoyé
// au navigateur une page tronquée, impossible à remplacer par une page
// d'erreur puisque le statut et une partie du corps seraient déjà partis.
// Avec le tampon, une erreur de rendu laisse la réponse intacte.
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
		// L'écriture a échoué alors que le statut est déjà parti : le
		// navigateur a probablement fermé la connexion. Il n'y a plus rien
		// à faire d'autre que de remonter l'erreur pour la journaliser.
		return fmt.Errorf("écriture de la réponse : %w", err)
	}

	return nil
}
