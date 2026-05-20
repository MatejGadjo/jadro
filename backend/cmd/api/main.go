package main

import (
	"net/http"

	"github.com/MatejGadjo/NewsAggregator/internal/app"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		app.Module,
		fx.Invoke(func(*http.Server) {}),
	).Run()
}
