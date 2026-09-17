// Package web embarque les gabarits HTML et les fichiers statiques dans le
// binaire.
//
// Le paquet ne contient que cette déclaration : une directive go:embed ne
// peut référencer que des fichiers situés dans le dossier du fichier source
// qui la porte. Les gabarits restent donc à côté du HTML et du CSS, à
// l'endroit où on s'attend à les trouver, et le paquet render n'a pas besoin
// de savoir où ils sont rangés.
//
// Conséquence pratique : le binaire compilé est autonome. Il fonctionne sans
// que le dossier web/ soit déployé à côté de lui.
package web

import "embed"

// Files contient les gabarits (templates/) et les fichiers statiques (static/).
//
// Le préfixe « all: » est nécessaire pour que go:embed inclue aussi les
// fichiers dont le nom commence par « _ » ou « . », qu'il ignore par défaut.
//
//go:embed all:templates static
var Files embed.FS
