package main

import (
	"log"

	"github.com/rohanthewiz/rterm/internal/app"
)

func main() {
	a := app.New()

	go func() {
		if err := a.Run(); err != nil {
			log.Fatal(err)
		}
		app.Exit(0)
	}()

	app.Main()
}
