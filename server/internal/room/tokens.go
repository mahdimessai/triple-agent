package room

type TokenDirectory struct {
	byPlayer map[string]string
	byToken  map[string]string
}

func NewTokenDirectory() *TokenDirectory {
	return &TokenDirectory{
		byPlayer: make(map[string]string),
		byToken:  make(map[string]string),
	}
}

func TokensFromMap(m map[string]string) *TokenDirectory {
	d := NewTokenDirectory()
	for playerID, token := range m {
		d.Register(playerID, token)
	}
	return d
}

func (d *TokenDirectory) Register(playerID, token string) {
	if token == "" {
		return
	}
	if oldToken, exists := d.byPlayer[playerID]; exists {
		delete(d.byToken, oldToken)
	}
	if oldPlayer, exists := d.byToken[token]; exists {
		delete(d.byPlayer, oldPlayer)
	}
	d.byPlayer[playerID] = token
	d.byToken[token] = playerID
}

func (d *TokenDirectory) Remove(playerID string) {
	token, exists := d.byPlayer[playerID]
	if !exists {
		return
	}
	delete(d.byPlayer, playerID)
	delete(d.byToken, token)
}

func (d *TokenDirectory) PlayerID(token string) (string, bool) {
	if token == "" {
		return "", false
	}
	playerID, ok := d.byToken[token]
	return playerID, ok
}

func (d *TokenDirectory) Authorize(playerID, token string) bool {
	if token == "" || playerID == "" {
		return false
	}
	expected, ok := d.byPlayer[playerID]
	return ok && expected == token
}
