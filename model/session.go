package model

const (
	SESSION_COOKIE_TOKEN  = "QAAUTHTOKEN"
	SESSION_COOKIE_USER   = "QAUSERID"
	SESSION_COOKIE_CSRF   = "QACSRF"
	SESSION_PROP_PLATFORM = "platform"
	SESSION_PROP_OS       = "os"
	SESSION_PROP_BROWSER  = "browser"
)

type Session struct {
	Id        string    `db:"Id, primarykey" json:"id"`
	Token     string    `db:"Token" json:"token"`
	CreateAt  int64     `db:"CreateAt" json:"create_at"`
	ExpiresAt int64     `db:"ExpiresAt" json:"expires_at"`
	UserId    string    `db:"UserId" json:"user_id"`
	Props     StringMap `db:"Props" json:"props"`
}

func (me *Session) PreSave() {
	if me.Id == "" {
		me.Id = NewId()
	}

	if me.Token == "" {
		me.Token = NewId()
	}

	me.CreateAt = GetMillis()

	if me.Props == nil {
		me.Props = make(map[string]string)
	}
}

func (me *Session) IsExpired() bool {
	if me.ExpiresAt <= 0 {
		return false
	}

	if GetMillis() > me.ExpiresAt {
		return true
	}

	return false
}

func (me *Session) SetExpireInDays(days int) {
	if me.CreateAt == 0 {
		me.ExpiresAt = GetMillis() + (1000 * 60 * 60 * 24 * int64(days))
	} else {
		me.ExpiresAt = me.CreateAt + (1000 * 60 * 60 * 24 * int64(days))
	}
}

func (me *Session) AddProp(key string, value string) {
	if me.Props == nil {
		me.Props = make(map[string]string)
	}

	me.Props[key] = value
}

func (me *Session) GenerateCSRF() string {
	token := NewId()
	me.AddProp("csrf", token)
	return token
}

func (me *Session) GetCSRF() string {
	if me.Props == nil {
		return ""
	}

	return me.Props["csrf"]
}
