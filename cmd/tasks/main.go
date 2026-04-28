package main

import (
	"log"

	"github.com/wysiwyg-git/tasks_v2/internal/app"
)

func main() {

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
