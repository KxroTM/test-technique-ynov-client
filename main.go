// Point d'entrée du client web.
//
// Le client est un serveur HTTP à part entière. Il ne contient aucune règle
// métier et n'accède jamais à la base de données : il rend des pages HTML et
// dialogue avec l'API du serveur.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KxroTM/test-technique-ynov-client/internal/api"
	"github.com/KxroTM/test-technique-ynov-client/internal/config"
	"github.com/KxroTM/test-technique-ynov-client/internal/handlers"
	"github.com/KxroTM/test-technique-ynov-client/internal/render"
	"github.com/KxroTM/test-technique-ynov-client/web"
)

func main() {
	// 1. Configuration.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration invalide : %v", err)
	}

	// 2. Gabarits. Ils sont compilés maintenant : une erreur de syntaxe
	// empêche le démarrage plutôt que d'apparaître au hasard d'une
	// navigation.
	renderer, err := render.New(web.Files)
	if err != nil {
		log.Fatalf("compilation des gabarits : %v", err)
	}

	staticHandler, err := renderer.StaticHandler()
	if err != nil {
		log.Fatalf("accès aux fichiers statiques : %v", err)
	}

	// 3. Client API et handlers.
	apiClient := api.NewClient(cfg.APIBaseURL)
	handler := handlers.New(apiClient, renderer)

	// 4. Démarrage du serveur.
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler.Routes(staticHandler),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("client web démarré sur http://localhost:%s", cfg.Port)
		log.Printf("API utilisée : %s", cfg.APIBaseURL)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("arrêt inattendu du serveur : %v", err)
		}
	}()

	// 5. Arrêt propre.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("arrêt du client en cours...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("arrêt forcé du serveur : %v", err)
	}
	log.Println("client arrêté")
}
