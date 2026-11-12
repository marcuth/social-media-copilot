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
	application.RegisterEvent[services.AIDelta](services.EventAIDelta)
	application.RegisterEvent[services.AIStatus](services.EventAIStatus)
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
	skillSvc := &services.SkillService{DB: conn}
	web := &services.WebService{DB: conn, Providers: providers}
	ai := &services.AIService{Profiles: profiles, Providers: providers, Skills: skillSvc, Web: web}
	chat := &services.ChatService{DB: conn, AI: ai, Posts: posts}
	backup := &services.BackupService{DB: conn, Path: path}

	app := application.New(application.Options{
		Name:        "Social Media Copilot",
		Description: "Agendamento e criação de conteúdo para redes sociais com IA",
		Services: []application.Service{
			application.NewService(&services.CatalogService{}),
			application.NewService(profiles),
			application.NewService(posts),
			application.NewService(providers),
			application.NewService(ai),
			application.NewService(skillSvc),
			application.NewService(chat),
			application.NewService(web),
			application.NewService(backup),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})
	emit := func(name string, data any) { app.Event.Emit(name, data) }
	ai.Emit = emit
	web.Emit = emit
	backup.Dialogs = dialogs{app}

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

type dialogs struct{ app *application.App }

func (d dialogs) SaveFile(title, defaultName string) (string, error) {
	return d.app.Dialog.SaveFile().
		SetMessage(title).
		SetFilename(defaultName).
		AddFilter("Banco SQLite", "*.db").
		PromptForSingleSelection()
}

func (d dialogs) OpenFile(title string) (string, error) {
	return d.app.Dialog.OpenFile().
		SetTitle(title).
		AddFilter("Banco SQLite", "*.db;*.sqlite").
		CanChooseFiles(true).
		PromptForSingleSelection()
}

func (d dialogs) Reveal(path string) error {
	return d.app.Browser.OpenFile(path)
}
