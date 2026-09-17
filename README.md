# Notes — Client web

Client web de l'application de gestion de notes par espaces, développé en Go
avec le moteur de gabarits `html/template`.

Ce dépôt contient **le client**. Le serveur API se trouve dans un dépôt
séparé : [test-technique-ynov](https://github.com/KxroTM/test-technique-ynov).

## Présentation

Le client est un **serveur HTTP à part entière**, et non une page statique. Il
génère les pages HTML côté serveur et dialogue avec l'API par requêtes HTTP.

```
Navigateur  ──HTML/formulaires──▶  Client Go  ──JSON/REST──▶  Serveur Go  ──SQL──▶  PostgreSQL
```

Trois points structurent cette architecture :

- Le client **ne contient aucune règle métier** et **n'accède jamais à la base
  de données**. Il ne connaît que l'adresse de l'API.
- Il ne connaît pas non plus le secret de signature des jetons JWT. Il reçoit
  un jeton à la connexion, le stocke dans un cookie et le rejoue vers l'API :
  il ne le vérifie pas lui-même, c'est le rôle du serveur.
- Le contrôle d'accès n'est **pas** réimplémenté ici. Le client affiche ce que
  l'API lui retourne ; si une ressource appartient à quelqu'un d'autre, l'API
  répond `404` et le client montre sa page « introuvable ». La sécurité ne
  dépend donc pas du client.

## Stack technique

| Composant       | Choix                | Raison |
|-----------------|----------------------|--------|
| Langage         | Go 1.25              | Imposé par le sujet |
| Gabarits        | `html/template`      | Imposé par le sujet ; échappement contextuel automatique |
| Routeur         | `net/http` (stdlib)  | Depuis Go 1.22, la stdlib gère méthode et paramètres d'URL |
| Style           | CSS écrit à la main  | Le sujet porte sur Go, pas sur le front |
| JavaScript      | Aucun (hors `confirm`) | L'application fonctionne entièrement sans JS |

**Ce dépôt n'a aucune dépendance externe.** Le fichier `go.mod` ne déclare
aucun `require` : tout repose sur la bibliothèque standard. Un routeur tiers
n'apportait rien pour une quinzaine de routes, et un framework CSS aurait
ajouté du bruit dans les gabarits sans répondre au sujet.

## Prérequis

- **Go 1.25** ou supérieur
- **Le serveur API démarré** (voir le README du dépôt serveur)

## Installation

### 1. Cloner le dépôt

```bash
git clone https://github.com/KxroTM/test-technique-ynov-client.git
cd test-technique-ynov-client
```

### 2. Configurer l'environnement

```bash
cp .env.example .env
```

Les valeurs par défaut conviennent à un environnement local : aucune
modification n'est nécessaire si le serveur écoute sur le port `8080`.

### 3. Lancer le client

```bash
go run ./cmd/web
```

L'application est accessible sur **http://localhost:3000**.

> Le serveur API doit tourner au préalable. Si ce n'est pas le cas, le client
> démarre quand même mais affiche un message d'indisponibilité : il n'a
> aucune raison de refuser de se lancer parce qu'un service distant est
> momentanément absent.

## Comptes de démonstration

| Email               | Mot de passe  |
|---------------------|---------------|
| `alice@example.com` | `password123` |
| `bob@example.com`   | `password123` |

Ces identifiants sont rappelés directement sur la page de connexion.

Connectez-vous avec les deux comptes tour à tour pour constater le
cloisonnement : les espaces et les notes de l'un ne sont jamais visibles
depuis l'autre, y compris en saisissant à la main l'URL d'une ressource
appartenant à l'autre compte.

## Fonctionnalités

| Écran | Adresse | Fonctionnalité |
|-------|---------|----------------|
| Connexion | `/login` | FT1 |
| Inscription | `/register` | FT1 |
| Liste des espaces | `/spaces` | FT2 |
| Création d'un espace | `/spaces/new` | FT2 |
| Modification d'un espace | `/spaces/{id}/edit` | FT2 |
| Détail d'un espace et ses notes | `/spaces/{id}` | FT2, FT3 |
| Ajout d'une note | `/spaces/{id}/notes/new` | FT4 |
| Modification d'une note | `/notes/{id}/edit` | FT5 |
| Suppression d'une note | `POST /notes/{id}/delete` | FT6 |

## Commandes utiles

| Commande       | Effet |
|----------------|-------|
| `make run`     | Lance le client |
| `make build`   | Compile le binaire dans `bin/` |
| `make test`    | Lance les tests |
| `make fmt`     | Formate le code |
| `make vet`     | Analyse statique |

Si `make` n'est pas disponible, les commandes `go` équivalentes s'utilisent
directement (`go run ./cmd/web`, `go test ./...`).

## Tests

```bash
go test ./...
```

**Aucune dépendance n'est requise pour les lancer** : ni serveur API, ni base
de données. Les tests montent l'application complète — routeur, handlers et
gabarits réels — devant une fausse API construite avec `httptest`. Ils
vérifient donc ce qu'un navigateur recevrait vraiment : codes de statut,
redirections, cookies et HTML produit.

Ce que la suite couvre :

| Domaine | Vérifications |
|---------|---------------|
| `internal/api` | Verbe et chemin de chaque appel, en-tête `Bearer`, corps JSON envoyé, classification des statuts, réponse `204` sans corps, corps d'erreur inattendu, API injoignable |
| `internal/session` | Attributs protecteurs du cookie (`HttpOnly`, `SameSite`), expiration alignée, cycle dépôt/relecture, suppression |
| `internal/render` | Découpe sur les runes et non les octets, coupe sur frontière de mot, formatage des dates |
| `internal/handlers` | Redirection de toutes les pages et actions protégées, connexion et déconnexion, échappement XSS, erreurs de validation, message de confirmation affiché une seule fois, API en panne sans perte de session |

Trois tests méritent d'être signalés :

- **`TestClientUsesExpectedMethodAndPath`** verrouille la traduction des
  actions : il vérifie qu'un `POST /spaces/7/edit` venu du navigateur part
  bien en `PUT /api/spaces/7` vers l'API.
- **`TestAPIFailureDoesNotDestroySession`** vérifie qu'une API en défaut
  affiche une page d'erreur *sans* déconnecter l'utilisateur, tandis que
  **`TestRejectedTokenClearsSessionAndRedirects`** vérifie qu'un jeton refusé
  provoque bien, lui, une déconnexion.
- **`TestNoteTitleIsEscaped`** injecte une note intitulée
  `<script>alert(1)</script>` et contrôle qu'elle est échappée **différemment**
  selon le contexte : en entités HTML dans le corps de la page, en séquences
  `\u` dans l'attribut JavaScript.

## Documentation technique

Le document `docs/documentation-technique.pdf` couvre l'ensemble de la solution
— serveur et client : choix techniques, architecture, modélisation, partis pris
d'implémentation et limites. Il est identique dans les deux dépôts.

Sa source HTML (`docs/documentation-technique.html`) est versionnée à côté du
PDF, afin de rester comparable d'une version à l'autre.

## Structure du projet

```
cmd/web/            Point d'entrée : assemblage et démarrage
internal/
  api/              Client HTTP typé vers le serveur (le seul à connaître l'API)
  config/           Configuration depuis l'environnement
  handlers/         Handlers de pages, routes, session requise
  render/           Compilation et exécution des gabarits
  session/          Cookie de session portant le jeton JWT
web/
  templates/        Layout, partials et pages
  static/           Feuille de style
  embed.go          Embarque templates/ et static/ dans le binaire
```

Le binaire est **autonome** : les gabarits et le CSS sont embarqués avec
`go:embed`. Il n'y a pas besoin de déployer le dossier `web/` à côté de
l'exécutable.

## Partis pris d'implémentation

### Les formulaires HTML ne savent faire que GET et POST

Un navigateur ne peut pas émettre de requête `PUT` ou `DELETE` depuis un
formulaire. Le client expose donc des routes `POST` explicites, et c'est le
client API qui les traduit dans le bon verbe REST :

| Action de l'utilisateur | Route du client | Appel vers l'API |
|-------------------------|-----------------|------------------|
| Modifier un espace | `POST /spaces/{id}/edit` | `PUT /api/spaces/{id}` |
| Supprimer un espace | `POST /spaces/{id}/delete` | `DELETE /api/spaces/{id}` |
| Modifier une note | `POST /notes/{id}/edit` | `PUT /api/notes/{id}` |
| Supprimer une note | `POST /notes/{id}/delete` | `DELETE /api/notes/{id}` |

L'API REST reste ainsi correcte, et l'application fonctionne sans une ligne de
JavaScript. La solution alternative — un champ caché `_method` interprété par
le serveur — aurait fait porter au serveur une contrainte propre au HTML.

Les suppressions sont en `POST` et non en `GET` pour une autre raison : un lien
`GET` peut être déclenché par le préchargement d'un navigateur ou par une
image distante. Une suppression partirait alors sans clic de l'utilisateur.

### Le jeton JWT est dans un cookie HttpOnly

Le client n'a pas de JavaScript pour garder un jeton en mémoire. Il le stocke
donc dans un cookie, dont les attributs sont choisis explicitement :

- `HttpOnly` : invisible depuis JavaScript, donc non exfiltrable en cas de XSS.
- `SameSite=Lax` : non envoyé lors de requêtes inter-sites, ce qui bloque les
  attaques CSRF sur les formulaires de modification et de suppression.
- `Expires` aligné sur l'expiration du jeton : le navigateur oublie le cookie
  au moment où le jeton devient invalide.

### « POST puis redirection » systématique

Toute action de modification répond par une redirection `303 See Other` et non
par une page. L'utilisateur peut ainsi actualiser sans rejouer l'action, et le
bouton « retour » ne propose pas de renvoyer le formulaire.

Le message de confirmation doit alors survivre à la redirection : il est porté
par un cookie de courte durée, lu et immédiatement effacé au rendu suivant.

### Les erreurs de saisie réaffichent le formulaire

Une erreur de validation n'est pas une panne. Le formulaire est réaffiché avec
les valeurs déjà saisies et les messages placés **sous les champs concernés**,
en s'appuyant sur le détail par champ que renvoie l'API. Le mot de passe, lui,
n'est jamais réaffiché.

Le code de statut de l'API est conservé (`400`, `401`, `409`) plutôt que
remplacé par un `200` : un formulaire réaffiché après un refus n'est pas un
succès.

### Session expirée et API injoignable sont deux cas distincts

C'est une distinction qui change le comportement ressenti :

- **Jeton refusé par l'API** (`401`) : la session est effacée et l'utilisateur
  est redirigé vers la connexion.
- **API injoignable** : une page d'erreur est affichée, mais **la session est
  conservée**. Déconnecter quelqu'un parce qu'un service distant est
  momentanément absent l'obligerait à se reconnecter sans raison.

### Les gabarits sont compilés au démarrage

Une erreur de syntaxe dans un gabarit fait échouer le lancement du serveur,
plutôt que d'apparaître au hasard d'une navigation. Le rendu passe en outre
par un tampon mémoire avant d'être écrit dans la réponse : si un gabarit
échoue à mi-parcours, le navigateur ne reçoit pas de page tronquée.

### L'échappement est assuré par html/template

`html/template` échappe automatiquement selon le contexte : le contenu d'une
note est échappé différemment dans du texte HTML, dans un attribut ou dans une
chaîne JavaScript. Une note intitulée `<script>alert(1)</script>` s'affiche
donc comme du texte, sans traitement particulier à écrire.

## Limites connues

- **`Secure` n'est pas activé sur le cookie de session**, car le
  développement se fait en HTTP. En production derrière HTTPS, cet attribut
  est indispensable.
- **Pas de déconnexion côté serveur.** Effacer le cookie suffit côté
  navigateur, mais le jeton reste techniquement valide jusqu'à son
  expiration : une API sans état ne peut pas révoquer un jeton déjà émis.
- **Pas de jeton anti-CSRF dédié.** La protection repose sur
  `SameSite=Lax`, qui couvre les navigateurs actuels. Un jeton par formulaire
  serait la défense complète.
- **Le profil est relu à chaque requête** (un appel à `/api/me`). C'est un
  aller-retour supplémentaire, assumé au profit de la simplicité : le jeton est
  ainsi toujours vérifié et le nom affiché toujours à jour.
