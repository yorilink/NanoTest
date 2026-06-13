package gate

import (
	"log"

	"github.com/lonng/nano/cluster/clusterpb"
	"github.com/lonng/nano/examples/gamecluster/store"
	"github.com/lonng/nano/session"
)

const (
	preferredGameServerKey = "preferredGameServerAddr"
	selectedGameServerKey  = "gameServerAddr"
)

type GameServerSelector struct {
	repo store.Repository
}

func NewGameServerSelector(repo store.Repository) *GameServerSelector {
	return &GameServerSelector{repo: repo}
}

func (s *GameServerSelector) Select(preferred string, members []*clusterpb.MemberInfo) (*clusterpb.MemberInfo, error) {
	if preferred != "" {
		for _, member := range members {
			if member.ServiceAddr == preferred {
				return member, nil
			}
		}
	}

	var selected *clusterpb.MemberInfo
	var selectedCount int64
	for _, member := range members {
		count, err := s.repo.GetOnlineCount(member.ServiceAddr)
		if err != nil {
			return nil, err
		}
		if selected == nil || count < selectedCount {
			selected = member
			selectedCount = count
		}
	}
	return selected, nil
}

func (s *GameServerSelector) Route(service string, session *session.Session, members []*clusterpb.MemberInfo) *clusterpb.MemberInfo {
	if service != "GameService" {
		if len(members) == 0 {
			return nil
		}
		return members[0]
	}

	preferred := session.String(preferredGameServerKey)
	member, err := s.Select(preferred, members)
	if err != nil {
		log.Println("select gameserver failed", err)
		return nil
	}
	if member != nil {
		session.Set(selectedGameServerKey, member.ServiceAddr)
	}
	return member
}
