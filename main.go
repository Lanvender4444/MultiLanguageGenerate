package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/Lanvender4444/MultiLanguageGenerate/ui"
)

func main() {
	a := app.New()
	application := ui.NewApp(a)
	application.Run()
}
