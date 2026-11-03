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

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options. The startup hook adjusts the initial
	// dimensions to the active display while keeping these safe fallback values.
	err := wails.Run(&options.App{
		Title:  "GoLock",
		Width:  820,
		Height: 520,

		// The window remains resizable. The startup hook applies screen-aware
		// minimum dimensions so the window can still fit smaller displays.
		MinWidth:  320,
		MinHeight: 360,

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
