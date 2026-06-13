package store

import "testing"

func TestMemoryRepositoryRoleLifecycle(t *testing.T) {
	repo := NewMemoryRepository()

	playerID, err := repo.NextPlayerID()
	if err != nil {
		t.Fatal(err)
	}
	role := &RoleSummary{
		AccountID:      10001,
		PlayerID:       playerID,
		Name:           "Alice",
		GameServerAddr: "127.0.0.1:34680",
	}
	if err := repo.CreateRole(role); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateRole(role); err != ErrRoleAlreadyExist {
		t.Fatalf("duplicate CreateRole error = %v, want ErrRoleAlreadyExist", err)
	}

	got, err := repo.GetRoleByAccount(10001)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlayerID != playerID || got.GameServerAddr != role.GameServerAddr {
		t.Fatalf("role = %+v, want %+v", got, role)
	}

	if err := repo.UpdateRoleGameServer(10001, playerID, "127.0.0.1:34681"); err != nil {
		t.Fatal(err)
	}
	got, err = repo.GetRoleByPlayer(playerID)
	if err != nil {
		t.Fatal(err)
	}
	if got.GameServerAddr != "127.0.0.1:34681" {
		t.Fatalf("GameServerAddr = %q", got.GameServerAddr)
	}
}

func TestMemoryRepositoryOnlineCount(t *testing.T) {
	repo := NewMemoryRepository()
	addr := "127.0.0.1:34680"

	if err := repo.IncOnlineCount(addr); err != nil {
		t.Fatal(err)
	}
	if err := repo.IncOnlineCount(addr); err != nil {
		t.Fatal(err)
	}
	if err := repo.DecOnlineCount(addr); err != nil {
		t.Fatal(err)
	}
	count, err := repo.GetOnlineCount(addr)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}
