package protocol

type LoginRequest struct {
	Token string `json:"token"`
}

type CreateRoleRequest struct {
	Token string `json:"token"`
	Name  string `json:"name"`
}

type PlayerSummary struct {
	AccountID      int64  `json:"accountId"`
	PlayerID       int64  `json:"playerId"`
	Name           string `json:"name"`
	GameServerAddr string `json:"gameServerAddr"`
}

type GateResponse struct {
	Code           int            `json:"code"`
	Message        string         `json:"message,omitempty"`
	NeedCreateRole bool           `json:"needCreateRole,omitempty"`
	RoleExisted    bool           `json:"roleExisted,omitempty"`
	Player         *PlayerSummary `json:"player,omitempty"`
}

type EnterRequest struct {
	AccountID      int64  `json:"accountId"`
	PlayerID       int64  `json:"playerId"`
	Name           string `json:"name"`
	GameServerAddr string `json:"gameServerAddr"`
}

type EnteredPush struct {
	Code           int    `json:"code"`
	Message        string `json:"message,omitempty"`
	AccountID      int64  `json:"accountId"`
	PlayerID       int64  `json:"playerId"`
	Name           string `json:"name"`
	GameServerAddr string `json:"gameServerAddr"`
	Reconnect      bool   `json:"reconnect"`
}

type PingRequest struct {
	Content string `json:"content"`
}

type PingResponse struct {
	PlayerID       int64  `json:"playerId"`
	GameServerAddr string `json:"gameServerAddr"`
	Content        string `json:"content"`
}
