package store

import "errors"

var (
	ErrRoleNotFound     = errors.New("role not found")
	ErrRoleAlreadyExist = errors.New("role already exists")
)

type RoleSummary struct {
	AccountID      int64
	PlayerID       int64
	Name           string
	GameServerAddr string
}

type OnlineLock struct {
	GateAddr        string
	SessionID       int64
	PlayerID        int64
	GameServerAddr  string
	LoginAtUnixTime int64
}

type Repository interface {
	NextPlayerID() (int64, error)
	GetRoleByAccount(accountID int64) (*RoleSummary, error)
	GetRoleByPlayer(playerID int64) (*RoleSummary, error)
	CreateRole(role *RoleSummary) error
	DeleteRole(accountID, playerID int64) error
	UpdateRoleGameServer(accountID, playerID int64, gameServerAddr string) error
	SetOnlineLock(accountID int64, lock *OnlineLock, ttlSeconds int64) error
	DeleteOnlineLock(accountID int64) error
	GetOnlineCount(gameServerAddr string) (int64, error)
	IncOnlineCount(gameServerAddr string) error
	DecOnlineCount(gameServerAddr string) error
}
