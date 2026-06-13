package game

import (
	"net"
	"testing"

	"github.com/lonng/nano/examples/gamecluster/protocol"
	"github.com/lonng/nano/examples/gamecluster/store"
	"github.com/lonng/nano/session"
)

type fakeEntity struct{}

func (fakeEntity) Push(string, interface{}) error        { return nil }
func (fakeEntity) RPC(string, interface{}) error         { return nil }
func (fakeEntity) LastMid() uint64                       { return 1 }
func (fakeEntity) Response(interface{}) error            { return nil }
func (fakeEntity) ResponseMid(uint64, interface{}) error { return nil }
func (fakeEntity) Close() error                          { return nil }
func (fakeEntity) RemoteAddr() net.Addr                  { return netAddr{} }

type netAddr struct{}

func (netAddr) Network() string { return "fake" }
func (netAddr) String() string  { return "fake" }

func TestGameServiceEnterReconnectDoesNotDoubleCount(t *testing.T) {
	repo := store.NewMemoryRepository()
	service := NewGameService(repo, "127.0.0.1:34680")
	s := session.New(fakeEntity{})

	req := &protocol.EnterRequest{
		AccountID: 10001,
		PlayerID:  50001,
		Name:      "Alice",
	}
	if err := service.Enter(s, req); err != nil {
		t.Fatal(err)
	}
	if err := service.Enter(s, req); err != nil {
		t.Fatal(err)
	}
	count, err := repo.GetOnlineCount("127.0.0.1:34680")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}

	service.OnSessionClosed(s)
	count, err = repo.GetOnlineCount("127.0.0.1:34680")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("count after close = %d, want 0", count)
	}
}
