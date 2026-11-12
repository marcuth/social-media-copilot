package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/marcuth/social-media-copilot/internal/db"
	"github.com/marcuth/social-media-copilot/internal/services"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Registering the event gives the frontend a typed listener for AI streaming.
	application.RegisterEvent[services.AIDelta](services.EventAIDelta)
}

func main() {
	path, err := db.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	conn, err := db.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	profiles := &services.ProfileService{DB: conn}
	posts := &services.PostService{DB: conn}
	providers := &services.ProviderService{DB: conn}
	ai := &services.AIService{Profiles: profiles, Providers: providers}

	app := application.New(application.Options{
		Name:        "Social Media Copilot",
		Description: "Agendamento e criação de conteúdo para redes sociais com IA",
		Services: []application.Service{
			application.NewService(&services.CatalogService{}),
			application.NewService(profiles),
			application.NewService(posts),
			application.NewService(providers),
			application.NewService(ai),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	ai.Emit = func(name string, data any) { app.Event.Emit(name, data) }

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Social Media Copilot",
		Width:            1360,
		Height:           860,
		MinWidth:         980,
		MinHeight:        640,
		StartState:       application.WindowStateMaximised,
		BackgroundColour: application.NewRGB(243, 244, 248),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
