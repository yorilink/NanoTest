package gate

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/lonng/nano/component"
	"github.com/lonng/nano/examples/gamecluster/auth"
	"github.com/lonng/nano/examples/gamecluster/protocol"
	"github.com/lonng/nano/examples/gamecluster/store"
	"github.com/lonng/nano/session"
)

const (
	sessionAccountIDKey = "accountId"
	sessionPlayerIDKey  = "playerId"
	sessionNameKey      = "name"
	onlineLockTTL       = int64(60)
)

type GateService struct {
	component.Base
	repo         store.Repository
	verifier     auth.TokenVerifier
	gateAddr     string
	localOnline  map[int64]*session.Session
	localOnlineM sync.Mutex
}

func NewGateService(repo store.Repository, verifier auth.TokenVerifier, gateAddr string) *GateService {
	return &GateService{
		repo:        repo,
		verifier:    verifier,
		gateAddr:    gateAddr,
		localOnline: map[int64]*session.Session{},
	}
}

func (gs *GateService) Login(s *session.Session, msg *protocol.LoginRequest) error {
	accountID, err := gs.verify(msg.Token)
	if err != nil {
		return s.Response(&protocol.GateResponse{Code: 401, Message: "invalid token"})
	}

	role, err := gs.repo.GetRoleByAccount(accountID)
	if err == store.ErrRoleNotFound {
		return s.Response(&protocol.GateResponse{Code: 0, NeedCreateRole: true})
	}
	if err != nil {
		log.Println("get role failed", err)
		return s.Response(&protocol.GateResponse{Code: 500, Message: "redis unavailable"})
	}

	return gs.enterGame(s, role, false)
}

func (gs *GateService) CreateRole(s *session.Session, msg *protocol.CreateRoleRequest) error {
	accountID, err := gs.verify(msg.Token)
	if err != nil {
		return s.Response(&protocol.GateResponse{Code: 401, Message: "invalid token"})
	}

	name := strings.TrimSpace(msg.Name)
	if name == "" {
		return s.Response(&protocol.GateResponse{Code: 400, Message: "role name required"})
	}

	role, err := gs.repo.GetRoleByAccount(accountID)
	if err == nil {
		return gs.enterGame(s, role, true)
	}
	if err != store.ErrRoleNotFound {
		log.Println("get role failed", err)
		return s.Response(&protocol.GateResponse{Code: 500, Message: "redis unavailable"})
	}

	playerID, err := gs.repo.NextPlayerID()
	if err != nil {
		log.Println("next player id failed", err)
		return s.Response(&protocol.GateResponse{Code: 500, Message: "redis unavailable"})
	}

	role = &store.RoleSummary{
		AccountID: accountID,
		PlayerID:  playerID,
		Name:      name,
	}
	createdRole := false
	if err := gs.repo.CreateRole(role); err != nil {
		if errors.Is(err, store.ErrRoleAlreadyExist) {
			existed, getErr := gs.repo.GetRoleByAccount(accountID)
			if getErr != nil {
				log.Println("get existing role failed", getErr)
				return s.Response(&protocol.GateResponse{Code: 500, Message: "redis unavailable"})
			}
			return gs.enterGame(s, existed, true)
		}
		log.Println("create role failed", err)
		return s.Response(&protocol.GateResponse{Code: 500, Message: "redis unavailable"})
	}
	createdRole = true

	return gs.enterGame(s, role, false, createdRole)
}

func (gs *GateService) OnSessionClosed(s *session.Session) {
	accountID := s.Int64(sessionAccountIDKey)
	if accountID < 1 {
		return
	}

	gs.localOnlineM.Lock()
	if gs.localOnline[accountID] == s {
		delete(gs.localOnline, accountID)
	}
	gs.localOnlineM.Unlock()

	if err := gs.repo.DeleteOnlineLock(accountID); err != nil {
		log.Println("delete online lock failed", err)
	}
}

func (gs *GateService) verify(token string) (int64, error) {
	if gs.verifier == nil {
		return 0, auth.ErrInvalidToken
	}
	return gs.verifier.Verify(token)
}

func (gs *GateService) enterGame(s *session.Session, role *store.RoleSummary, roleExisted bool, createdRole ...bool) error {
	if err := s.Bind(role.PlayerID); err != nil {
		return s.Response(&protocol.GateResponse{Code: 500, Message: "bind player failed"})
	}

	s.Set(sessionAccountIDKey, role.AccountID)
	s.Set(sessionPlayerIDKey, role.PlayerID)
	s.Set(sessionNameKey, role.Name)
	s.Set(preferredGameServerKey, role.GameServerAddr)
	s.Set(selectedGameServerKey, "")

	enterReq := &protocol.EnterRequest{
		AccountID:      role.AccountID,
		PlayerID:       role.PlayerID,
		Name:           role.Name,
		GameServerAddr: role.GameServerAddr,
	}
	if err := s.RPC("GameService.Enter", enterReq); err != nil {
		log.Println("enter gameserver failed", err)
		s.Router().Delete("GameService")
		gs.rollbackCreatedRole(role, createdRole...)
		return s.Response(&protocol.GateResponse{Code: 503, Message: "no available gameserver"})
	}

	gameServerAddr := s.String(selectedGameServerKey)
	if gameServerAddr == "" {
		s.Router().Delete("GameService")
		gs.rollbackCreatedRole(role, createdRole...)
		return s.Response(&protocol.GateResponse{Code: 503, Message: "no available gameserver"})
	}
	if gameServerAddr != role.GameServerAddr {
		if err := gs.repo.UpdateRoleGameServer(role.AccountID, role.PlayerID, gameServerAddr); err != nil {
			log.Println("update role gameserver failed", err)
			s.Router().Delete("GameService")
			return s.Response(&protocol.GateResponse{Code: 500, Message: "redis unavailable"})
		}
		role.GameServerAddr = gameServerAddr
	}

	gs.replaceLocalSession(role.AccountID, s)
	if err := gs.repo.SetOnlineLock(role.AccountID, &store.OnlineLock{
		GateAddr:        gs.gateAddr,
		SessionID:       s.ID(),
		PlayerID:        role.PlayerID,
		GameServerAddr:  gameServerAddr,
		LoginAtUnixTime: time.Now().Unix(),
	}, onlineLockTTL); err != nil {
		log.Println("set online lock failed", err)
		s.Router().Delete("GameService")
		return s.Response(&protocol.GateResponse{Code: 500, Message: "redis unavailable"})
	}

	return s.Response(&protocol.GateResponse{
		Code:        0,
		RoleExisted: roleExisted,
		Player: &protocol.PlayerSummary{
			AccountID:      role.AccountID,
			PlayerID:       role.PlayerID,
			Name:           role.Name,
			GameServerAddr: gameServerAddr,
		},
	})
}

func (gs *GateService) rollbackCreatedRole(role *store.RoleSummary, createdRole ...bool) {
	if len(createdRole) == 0 || !createdRole[0] {
		return
	}
	if err := gs.repo.DeleteRole(role.AccountID, role.PlayerID); err != nil {
		log.Println("rollback role failed", err)
	}
}

func (gs *GateService) replaceLocalSession(accountID int64, s *session.Session) {
	gs.localOnlineM.Lock()
	old := gs.localOnline[accountID]
	gs.localOnline[accountID] = s
	gs.localOnlineM.Unlock()

	if old != nil && old != s {
		old.Close()
	}
}
