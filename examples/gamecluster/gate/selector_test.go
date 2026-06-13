package gate

import (
	"testing"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/examples/gamecluster/store"
)

func TestGameServerSelectorPrefersAvailableOldServer(t *testing.T) {
	repo := store.NewMemoryRepository()
	selector := NewGameServerSelector(repo)
	members := []*clusterpb.MemberInfo{
		{ServiceAddr: "127.0.0.1:1"},
		{ServiceAddr: "127.0.0.1:2"},
	}

	selected, err := selector.Select("127.0.0.1:2", members)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ServiceAddr != "127.0.0.1:2" {
		t.Fatalf("selected = %s", selected.ServiceAddr)
	}
}

func TestGameServerSelectorFallsBackToLowestOnlineCount(t *testing.T) {
	repo := store.NewMemoryRepository()
	selector := NewGameServerSelector(repo)
	if err := repo.IncOnlineCount("127.0.0.1:1"); err != nil {
		t.Fatal(err)
	}
	members := []*clusterpb.MemberInfo{
		{ServiceAddr: "127.0.0.1:1"},
		{ServiceAddr: "127.0.0.1:2"},
	}

	selected, err := selector.Select("127.0.0.1:dead", members)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ServiceAddr != "127.0.0.1:2" {
		t.Fatalf("selected = %s", selected.ServiceAddr)
	}
}

func TestGameServerSelectorReturnsNilWithoutMembers(t *testing.T) {
	repo := store.NewMemoryRepository()
	selector := NewGameServerSelector(repo)

	selected, err := selector.Select("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected != nil {
		t.Fatalf("selected = %+v, want nil", selected)
	}
}
