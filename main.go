package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// Tamanho fixo da janela (o mesmo em qualquer tela).
const (
	windowWidth  = 460
	windowHeight = 780
)

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "GoLock",
		Width:  windowWidth,
		Height: windowHeight,

		DisableResize: true,
		MinWidth:      windowWidth,
		MinHeight:     windowHeight,
		MaxWidth:      windowWidth,
		MaxHeight:     windowHeight,

		BackgroundColour: &options.RGBA{R: 14, G: 10, B: 26, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown, // apaga a chave da memória ao fechar
		Bind: []interface{}{
			app,
		},

		Windows: &windows.Options{
			IsZoomControlEnabled: false,
			DisablePinchZoom:     true,
		},
	})

	if err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}
