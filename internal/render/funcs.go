package render

import (
	"html/template"
	"time"
)

// templateFuncs retourne les fonctions utilisables depuis les gabarits.
//
// Le jeu est volontairement réduit : la mise en forme appartient aux
// gabarits, mais la logique appartient au code Go. On ne trouvera donc ici
// que de la présentation pure, jamais de règle métier.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"formatDate": formatDate,
		"truncate":   truncate,
	}
}

// formatDate met une date au format jour/mois/année suivi de l'heure.
//
// Le format Go est écrit avec la date de référence « 02/01/2006 15:04 », qui
// correspond à 2006-01-02 15:04:05 : c'est la convention de la bibliothèque
// standard, où chaque composant a un numéro fixe.
func formatDate(value time.Time) string {
	return value.Local().Format("02/01/2006 à 15:04")
}

// truncate raccourcit un texte à la longueur demandée, en ajoutant une
// ellipse si la coupe a lieu.
//
// La découpe se fait sur les runes et non sur les octets : couper une chaîne
// UTF-8 au milieu d'un caractère accentué produirait un octet invalide affiché
// comme un losange noir.
//
// La coupe est en outre reculée jusqu'au dernier espace afin de ne pas
// tronquer un mot en son milieu.
func truncate(text string, maxLength int) string {
	runes := []rune(text)
	if len(runes) <= maxLength {
		return text
	}

	cut := runes[:maxLength]

	for i := len(cut) - 1; i >= 0; i-- {
		if cut[i] == ' ' {
			cut = cut[:i]
			break
		}
	}

	return string(cut) + "…"
}
