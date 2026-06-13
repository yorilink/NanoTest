package game

import (
	"log"
	"sync"

	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/gamecluster/protocol"
	"github.com/lonng/nano/examples/gamecluster/store"
	"github.com/lonng/nano/session"
)

const (
	sessionAccountIDKey = "accountId"
	sessionPlayerIDKey  = "playerId"
	sessionNameKey      = "name"
	sessionGameAddrKey  = "gameServerAddr"
)

type onlinePlayer struct {
	accountID int64
	playerID  int64
	name      string
	session   *session.Session
}

type GameService struct {
	component.Base
	repo        store.Repository
	serviceAddr string
	mu          sync.Mutex
	players     map[int64]*onlinePlayer
}

func NewGameService(repo store.Repository, serviceAddr string) *GameService {
	return &GameService{
		repo:        repo,
		serviceAddr: serviceAddr,
		players:     map[int64]*onlinePlayer{},
	}
}

func (gs *GameService) Enter(s *session.Session, msg *protocol.EnterRequest) error {
	reconnect := false

	gs.mu.Lock()
	if old, ok := gs.players[msg.PlayerID]; ok {
		reconnect = true
		old.session = s
		old.accountID = msg.AccountID
		old.name = msg.Name
	} else {
		gs.players[msg.PlayerID] = &onlinePlayer{
			accountID: msg.AccountID,
			playerID:  msg.PlayerID,
			name:      msg.Name,
			session:   s,
		}
	}
	gs.mu.Unlock()

	if !reconnect {
		if err := gs.repo.IncOnlineCount(gs.serviceAddr); err != nil {
			log.Println("increment online count failed", err)
		}
	}

	if err := s.Bind(msg.PlayerID); err != nil {
		return err
	}
	s.Set(sessionAccountIDKey, msg.AccountID)
	s.Set(sessionPlayerIDKey, msg.PlayerID)
	s.Set(sessionNameKey, msg.Name)
	s.Set(sessionGameAddrKey, gs.serviceAddr)

	return s.Push("GameService.Entered", &protocol.EnteredPush{
		Code:           0,
		AccountID:      msg.AccountID,
		PlayerID:       msg.PlayerID,
		Name:           msg.Name,
		GameServerAddr: gs.serviceAddr,
		Reconnect:      reconnect,
	})
}

func (gs *GameService) Ping(s *session.Session, msg *protocol.PingRequest) error {
	return s.Response(&protocol.PingResponse{
		PlayerID:       s.UID(),
		GameServerAddr: gs.serviceAddr,
		Content:        msg.Content,
	})
}

func (gs *GameService) OnSessionClosed(s *session.Session) {
	playerID := s.UID()
	if playerID < 1 {
		return
	}

	removed := false
	gs.mu.Lock()
	if player, ok := gs.players[playerID]; ok && player.session == s {
		delete(gs.players, playerID)
		removed = true
	}
	gs.mu.Unlock()

	if removed {
		if err := gs.repo.DecOnlineCount(gs.serviceAddr); err != nil {
			log.Println("decrement online count failed", err)
		}
	}
}
