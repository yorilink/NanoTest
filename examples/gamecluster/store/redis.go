package store

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type RedisRepository struct {
	addr    string
	timeout time.Duration
}

func NewRedisRepository(addr string) *RedisRepository {
	return &RedisRepository{
		addr:    addr,
		timeout: 3 * time.Second,
	}
}

func (r *RedisRepository) NextPlayerID() (int64, error) {
	return r.intCommand("INCR", keyPlayerID())
}

func (r *RedisRepository) GetRoleByAccount(accountID int64) (*RoleSummary, error) {
	values, err := r.hashCommand("HGETALL", keyAccountPlayer(accountID))
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, ErrRoleNotFound
	}
	return roleFromHash(accountID, values)
}

func (r *RedisRepository) GetRoleByPlayer(playerID int64) (*RoleSummary, error) {
	values, err := r.hashCommand("HGETALL", keyPlayerProfile(playerID))
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, ErrRoleNotFound
	}
	role, err := roleFromHash(0, values)
	if err != nil {
		return nil, err
	}
	if role.PlayerID == 0 {
		role.PlayerID = playerID
	}
	return role, nil
}

func (r *RedisRepository) CreateRole(role *RoleSummary) error {
	if _, err := r.GetRoleByAccount(role.AccountID); err == nil {
		return ErrRoleAlreadyExist
	} else if err != ErrRoleNotFound {
		return err
	}

	if _, err := r.command("HMSET", keyAccountPlayer(role.AccountID),
		"playerId", strconv.FormatInt(role.PlayerID, 10),
		"name", role.Name,
		"gameServerAddr", role.GameServerAddr,
	); err != nil {
		return err
	}
	_, err := r.command("HMSET", keyPlayerProfile(role.PlayerID),
		"accountId", strconv.FormatInt(role.AccountID, 10),
		"playerId", strconv.FormatInt(role.PlayerID, 10),
		"name", role.Name,
		"gameServerAddr", role.GameServerAddr,
	)
	return err
}

func (r *RedisRepository) DeleteRole(accountID, playerID int64) error {
	_, err := r.command("DEL", keyAccountPlayer(accountID), keyPlayerProfile(playerID))
	return err
}

func (r *RedisRepository) UpdateRoleGameServer(accountID, playerID int64, gameServerAddr string) error {
	if _, err := r.command("HMSET", keyAccountPlayer(accountID), "gameServerAddr", gameServerAddr); err != nil {
		return err
	}
	_, err := r.command("HMSET", keyPlayerProfile(playerID), "gameServerAddr", gameServerAddr)
	return err
}

func (r *RedisRepository) SetOnlineLock(accountID int64, lock *OnlineLock, ttlSeconds int64) error {
	key := keyOnlineAccount(accountID)
	if _, err := r.command("HMSET", key,
		"gateAddr", lock.GateAddr,
		"sessionId", strconv.FormatInt(lock.SessionID, 10),
		"playerId", strconv.FormatInt(lock.PlayerID, 10),
		"gameServerAddr", lock.GameServerAddr,
		"loginAt", strconv.FormatInt(lock.LoginAtUnixTime, 10),
	); err != nil {
		return err
	}
	if ttlSeconds > 0 {
		_, err := r.command("EXPIRE", key, strconv.FormatInt(ttlSeconds, 10))
		return err
	}
	return nil
}

func (r *RedisRepository) DeleteOnlineLock(accountID int64) error {
	_, err := r.command("DEL", keyOnlineAccount(accountID))
	return err
}

func (r *RedisRepository) GetOnlineCount(gameServerAddr string) (int64, error) {
	value, err := r.command("GET", keyGameServerOnline(gameServerAddr))
	if err != nil {
		return 0, err
	}
	if value == nil {
		return 0, nil
	}
	raw, ok := value.(string)
	if !ok || raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

func (r *RedisRepository) IncOnlineCount(gameServerAddr string) error {
	_, err := r.command("INCR", keyGameServerOnline(gameServerAddr))
	return err
}

func (r *RedisRepository) DecOnlineCount(gameServerAddr string) error {
	count, err := r.intCommand("DECR", keyGameServerOnline(gameServerAddr))
	if err != nil {
		return err
	}
	if count < 0 {
		_, err = r.command("SET", keyGameServerOnline(gameServerAddr), "0")
	}
	return err
}

func (r *RedisRepository) intCommand(args ...string) (int64, error) {
	value, err := r.command(args...)
	if err != nil {
		return 0, err
	}
	switch v := value.(type) {
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("redis command %s returned %T", args[0], value)
	}
}

func (r *RedisRepository) hashCommand(args ...string) (map[string]string, error) {
	value, err := r.command(args...)
	if err != nil {
		return nil, err
	}
	items, ok := value.([]string)
	if !ok {
		return nil, fmt.Errorf("redis command %s returned %T", args[0], value)
	}
	result := make(map[string]string, len(items)/2)
	for i := 0; i+1 < len(items); i += 2 {
		result[items[i]] = items[i+1]
	}
	return result, nil
}

func (r *RedisRepository) command(args ...string) (interface{}, error) {
	conn, err := net.DialTimeout("tcp", r.addr, r.timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(r.timeout)); err != nil {
		return nil, err
	}
	if _, err := conn.Write(encodeRESP(args)); err != nil {
		return nil, err
	}
	return readRESP(bufio.NewReader(conn))
}

func encodeRESP(args []string) []byte {
	var b strings.Builder
	b.WriteString("*")
	b.WriteString(strconv.Itoa(len(args)))
	b.WriteString("\r\n")
	for _, arg := range args {
		b.WriteString("$")
		b.WriteString(strconv.Itoa(len(arg)))
		b.WriteString("\r\n")
		b.WriteString(arg)
		b.WriteString("\r\n")
	}
	return []byte(b.String())
}

func readRESP(r *bufio.Reader) (interface{}, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch prefix {
	case '+':
		return readLine(r)
	case '-':
		line, _ := readLine(r)
		return nil, fmt.Errorf("redis error: %s", line)
	case ':':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		return strconv.ParseInt(line, 10, 64)
	case '$':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, nil
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		return string(buf[:n]), nil
	case '*':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return []string{}, nil
		}
		items := make([]string, 0, n)
		for i := 0; i < n; i++ {
			value, err := readRESP(r)
			if err != nil {
				return nil, err
			}
			if value == nil {
				items = append(items, "")
				continue
			}
			items = append(items, fmt.Sprint(value))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unexpected redis response prefix %q", prefix)
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func roleFromHash(accountID int64, values map[string]string) (*RoleSummary, error) {
	playerID, err := strconv.ParseInt(values["playerId"], 10, 64)
	if err != nil {
		return nil, err
	}
	if accountID == 0 && values["accountId"] != "" {
		accountID, err = strconv.ParseInt(values["accountId"], 10, 64)
		if err != nil {
			return nil, err
		}
	}
	return &RoleSummary{
		AccountID:      accountID,
		PlayerID:       playerID,
		Name:           values["name"],
		GameServerAddr: values["gameServerAddr"],
	}, nil
}

func keyPlayerID() string {
	return "gamecluster:player_id"
}

func keyAccountPlayer(accountID int64) string {
	return fmt.Sprintf("gamecluster:account:%d:player", accountID)
}

func keyPlayerProfile(playerID int64) string {
	return fmt.Sprintf("gamecluster:player:%d:profile", playerID)
}

func keyOnlineAccount(accountID int64) string {
	return fmt.Sprintf("gamecluster:online:account:%d", accountID)
}

func keyGameServerOnline(addr string) string {
	return fmt.Sprintf("gamecluster:gameserver:%s:online_count", addr)
}
