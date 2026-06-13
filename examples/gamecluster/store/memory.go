package store

import "sync"

type MemoryRepository struct {
	mu           sync.Mutex
	nextPlayerID int64
	byAccount    map[int64]*RoleSummary
	byPlayer     map[int64]*RoleSummary
	onlineLocks  map[int64]*OnlineLock
	onlineCounts map[string]int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byAccount:    map[int64]*RoleSummary{},
		byPlayer:     map[int64]*RoleSummary{},
		onlineLocks:  map[int64]*OnlineLock{},
		onlineCounts: map[string]int64{},
	}
}

func (r *MemoryRepository) NextPlayerID() (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextPlayerID++
	return r.nextPlayerID, nil
}

func (r *MemoryRepository) GetRoleByAccount(accountID int64) (*RoleSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	role, ok := r.byAccount[accountID]
	if !ok {
		return nil, ErrRoleNotFound
	}
	return cloneRole(role), nil
}

func (r *MemoryRepository) GetRoleByPlayer(playerID int64) (*RoleSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	role, ok := r.byPlayer[playerID]
	if !ok {
		return nil, ErrRoleNotFound
	}
	return cloneRole(role), nil
}

func (r *MemoryRepository) CreateRole(role *RoleSummary) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byAccount[role.AccountID]; ok {
		return ErrRoleAlreadyExist
	}
	copied := cloneRole(role)
	r.byAccount[role.AccountID] = copied
	r.byPlayer[role.PlayerID] = cloneRole(role)
	return nil
}

func (r *MemoryRepository) DeleteRole(accountID, playerID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.byAccount, accountID)
	delete(r.byPlayer, playerID)
	return nil
}

func (r *MemoryRepository) UpdateRoleGameServer(accountID, playerID int64, gameServerAddr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	accountRole, ok := r.byAccount[accountID]
	if !ok {
		return ErrRoleNotFound
	}
	playerRole, ok := r.byPlayer[playerID]
	if !ok {
		return ErrRoleNotFound
	}
	accountRole.GameServerAddr = gameServerAddr
	playerRole.GameServerAddr = gameServerAddr
	return nil
}

func (r *MemoryRepository) SetOnlineLock(accountID int64, lock *OnlineLock, _ int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	copied := *lock
	r.onlineLocks[accountID] = &copied
	return nil
}

func (r *MemoryRepository) DeleteOnlineLock(accountID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.onlineLocks, accountID)
	return nil
}

func (r *MemoryRepository) GetOnlineCount(gameServerAddr string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.onlineCounts[gameServerAddr], nil
}

func (r *MemoryRepository) IncOnlineCount(gameServerAddr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.onlineCounts[gameServerAddr]++
	return nil
}

func (r *MemoryRepository) DecOnlineCount(gameServerAddr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.onlineCounts[gameServerAddr] > 0 {
		r.onlineCounts[gameServerAddr]--
	}
	return nil
}

func cloneRole(role *RoleSummary) *RoleSummary {
	if role == nil {
		return nil
	}
	copied := *role
	return &copied
}
