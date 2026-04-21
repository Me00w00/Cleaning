package main

import (
	"log"

	"project_cleaning/internal/platform/bootstrap"
)

func main() {
	if err := bootstrap.Run(); err != nil {
		log.Fatal(err)
	}
}
