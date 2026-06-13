package gate

import (
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/gamecluster/auth"
	"github.com/lonng/nano/examples/gamecluster/store"
	"github.com/lonng/nano/session"
)

var (
	Services     = &component.Components{}
	gateService  *GateService
	routeService *GameServerSelector
)

func Init(repo store.Repository, gateAddr string) {
	Services = &component.Components{}
	gateService = NewGateService(repo, auth.DemoTokenVerifier{}, gateAddr)
	routeService = NewGameServerSelector(repo)
	Services.Register(gateService)
}

func RouteService() *GameServerSelector {
	return routeService
}

func OnSessionClosed(s *session.Session) {
	if gateService != nil {
		gateService.OnSessionClosed(s)
	}
}
