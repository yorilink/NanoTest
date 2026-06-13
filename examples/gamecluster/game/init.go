package game

import (
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/gamecluster/store"
	"github.com/lonng/nano/session"
)

var (
	Services    = &component.Components{}
	gameService *GameService
)

func Init(repo store.Repository, serviceAddr string) {
	Services = &component.Components{}
	gameService = NewGameService(repo, serviceAddr)
	Services.Register(gameService)
}

func OnSessionClosed(s *session.Session) {
	if gameService != nil {
		gameService.OnSessionClosed(s)
	}
}
