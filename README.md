# Notes : client web

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
| Langage         | Go 1.25              | Même langage que le serveur, binaire autonome |
| Gabarits        | `html/template`      | Rendu côté serveur, échappement contextuel automatique |
| Routeur         | `net/http` (stdlib)  | Depuis Go 1.22, la stdlib gère méthode et paramètres d'URL |
| Style           | CSS écrit à la main  | Système de tokens, thèmes clair et sombre |
| Police          | Inter, hébergée localement | Substitution à SF Pro, aucune requête vers un tiers |
| JavaScript      | Aucune bibliothèque | Glisser-déposer et fenêtre de confirmation, en enrichissement seul |

**Ce dépôt n'a aucune dépendance externe.** Le fichier `go.mod` ne déclare
aucun `require` : tout repose sur la bibliothèque standard. Un routeur tiers
n'apportait rien pour une quinzaine de routes, et un framework CSS aurait
alourdi les gabarits pour un résultat que le CSS écrit à la main atteint
directement.

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


### 3. Activer la connexion Google (facultatif)

Renseigner dans `.env` l'identifiant public du client OAuth, le même que celui
du serveur :

```
GOOGLE_CLIENT_ID=...
```

Sans cette variable, le bouton n'apparaît pas. Le secret, lui, ne concerne que
le serveur et ne doit jamais figurer ici.

### 4. Lancer le client

```bash
go run .
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

| Écran | Adresse |
|-------|---------|
| Connexion | `/login` |
| Inscription | `/register` |
| Connexion Google | `/auth/google` |
| Liste des espaces | `/spaces` |
| Création d'un espace | `/spaces/new` |
| Modification d'un espace | `/spaces/{id}/edit` |
| Board d'un espace | `/spaces/{id}` |
| Ajout d'une note | `/spaces/{id}/notes/new` |
| Modification d'une note | `/notes/{id}/edit` |
| Changement d'état d'une note | `POST /notes/{id}/status` |
| Suppression d'une note | `POST /notes/{id}/delete` |

Les notes d'un espace ne sont pas présentées en liste mais en **board** : une
colonne par état, une carte par note. Déplacer une carte change son état.

## Commandes utiles

| Commande       | Effet |
|----------------|-------|
| `make run`     | Lance le client |
| `make build`   | Compile le binaire dans `bin/` |
| `make fmt`     | Formate le code |
| `make vet`     | Analyse statique |

Si `make` n'est pas disponible, les commandes `go` équivalentes s'utilisent
directement (`go run .`, `go build ./...`).

## Vérifications

Les contrôles suivants ont été menés dans le navigateur et par requêtes
directes :

| Contrôle | Résultat attendu |
|----------|------------------|
| Toute page ou action protégée sans session | Redirection vers `/login` |
| Jeton refusé par l'API | Session effacée, retour à la connexion |
| API injoignable | Page d'erreur, **session conservée** |
| Ressource appartenant à un autre compte | Page « introuvable » |
| Identifiant d'URL invalide | Page « introuvable » |
| Note dont le titre contient une balise | Échappée, jamais exécutée |
| Erreur de saisie | Formulaire réaffiché, valeurs conservées, message sous le champ |
| Déplacement d'une note sans JavaScript | Flèches de la carte, formulaire classique |
| Déplacement par glisser avec JavaScript | Colonnes, compteurs et jauge mis à jour |
| Échec réseau pendant un glisser | Carte remise en place et message affiché |

```bash
go build ./...
go vet ./...
```

## Documentation technique

Le document `docs/documentation-technique.pdf` couvre l'ensemble de la solution,
serveur et client : choix techniques, architecture, modélisation, partis pris
d'implémentation et limites. Il est identique dans les deux dépôts.

Sa source HTML (`docs/documentation-technique.html`) est versionnée à côté du
PDF, afin de rester comparable d'une version à l'autre.

## Structure du projet

```
main.go             Point d'entrée : assemblage et démarrage
internal/
  api/              Client HTTP typé vers le serveur (le seul à connaître l'API)
  config/           Configuration depuis l'environnement
  handlers/         Handlers de pages, routes, session requise, flux Google
  render/           Compilation et exécution des gabarits
  session/          Cookie de session portant le jeton JWT
web/
  templates/        Layout, partials et pages
  static/
    style.css       Système de design : tokens, thèmes clair et sombre
    board.js        Glisser-déposer des notes entre colonnes
    confirm.js      Fenêtre de confirmation avant suppression
    fonts/          Police Inter, hébergée dans le projet
  embed.go          Embarque templates/ et static/ dans le binaire
```

Le binaire est **autonome** : gabarits, CSS, scripts et police sont embarqués
avec `go:embed`. Il n'y a pas besoin de déployer le dossier `web/` à côté de
l'exécutable, ni d'appeler un service tiers au chargement des pages.

## Partis pris d'implémentation

### Le board, et le déplacement sans JavaScript

Les trois états d'une note correspondent exactement à trois colonnes. Déplacer
une carte change son état, par deux chemins qui aboutissent au même résultat :

| Moyen | Mécanisme |
|-------|-----------|
| Sans JavaScript | Deux flèches sur la carte, formulaire `POST`, redirection `303` |
| Avec JavaScript | Glisser-déposer, `fetch` en arrière-plan, réponse `204` |

Le serveur rend déjà les flèches dans le bon sens : une carte « Non fait »
masque sa flèche gauche, une carte « Terminé » masque sa droite. Le script ne
contient **aucun libellé ni ordre d'états codé en dur**, il les lit dans le HTML
produit par Go. Si l'appel échoue, la carte revient à sa place, la jauge
d'avancement est restaurée et un message apparaît.

### Une fenêtre de confirmation plutôt que celle du navigateur

Les suppressions passent par l'élément natif `<dialog>`, qui apporte le
piégeage du focus, la touche Échap et l'inertisation de l'arrière-plan sans
avoir à les réécrire. Le texte reste dans les gabarits, porté par des attributs
`data-confirm` ; le script est générique et ne connaît aucun libellé.

Sur un navigateur sans support de `<dialog>`, le formulaire s'envoie
directement : la suppression reste possible plutôt que d'être silencieusement
bloquée.

### Les messages s'effacent seuls, en CSS

Un message de confirmation disparaît au bout de quatre secondes par une
animation CSS, et non par un minuteur JavaScript : le comportement est donc le
même avec ou sans script. Le survol suspend le retrait, le temps de finir de
lire. L'animation replie aussi la hauteur et les marges, sans quoi un espace
vide resterait dans la page.

### Un système de design plutôt qu'une feuille de style

Le CSS repose sur une centaine de variables décrivant couleurs, échelle
typographique, espacements, rayons et courbes d'animation. Les composants ne
référencent que ces variables, jamais une valeur littérale. Le **thème sombre**
se limite donc à redéfinir les variables sous `prefers-color-scheme`.

La police **Inter** est hébergée dans le projet et embarquée dans le binaire.
Elle sert de substitution à SF Pro, qui n'est pas redistribuable sur le web.
Aucune requête n'est faite vers un service tiers, et l'application reste
identique hors ligne.

Toute animation est neutralisée sous `prefers-reduced-motion`.

### Connexion Google

Le client redirige vers Google avec le `client_id`, qui est public, mais c'est
**l'API qui possède l'authentification** : elle seule détient le `client_secret`,
échange le code et émet le JWT. Un paramètre `state` aléatoire, déposé en cookie
et comparé en temps constant au retour, garantit que la réponse de Google
correspond à une demande réellement initiée depuis ce navigateur.

Sans identifiants configurés, le bouton n'apparaît pas et la connexion par mot
de passe reste inchangée.

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
JavaScript. La solution alternative (un champ caché `_method` interprété par
le serveur) aurait fait porter au serveur une contrainte propre au HTML.

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

- **La connexion Google n'est pas testable sans identifiants OAuth.** Elle exige
  un projet Google Cloud propre à celui qui l'exécute. Sans les variables
  correspondantes, le bouton n'apparaît pas et l'application reste entièrement
  utilisable par email et mot de passe.
- **Le glisser-déposer demande un pointeur.** Au clavier ou sur mobile, le
  déplacement se fait par les flèches de la carte, qui restent le chemin de
  référence.
- **Sans JavaScript, la suppression ne demande pas de confirmation.** Une page
  de confirmation rendue par le serveur serait la réponse complète.
