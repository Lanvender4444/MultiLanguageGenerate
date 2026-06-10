package main

import (
	"fyne.io/fyne/v2/app"

	"github.com/yourname/MultiLanguageGenerate/ui"
)

func main() {
	a := app.New()
	application := ui.NewApp(a)
	application.Run()
}
