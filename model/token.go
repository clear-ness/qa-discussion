package model

import (
	"net/http"
)

const (
	TOKEN_SIZE = 64
)

type Token struct {
	Token    string `db:"Token, primarykey"`
	CreateAt int64  `db:"CreateAt"`
	Type     string `db:"Type"`
	Extra    string `db:"Extra"`
}

func NewToken(tokentype, extra string) *Token {
	return &Token{
		Token:    NewRandomString(TOKEN_SIZE),
		CreateAt: GetMillis(),
		Type:     tokentype,
		Extra:    extra,
	}
}

func (t *Token) IsValid() *AppError {
	if len(t.Token) != TOKEN_SIZE {
		return NewAppError("Token.IsValid", "model.token.is_valid.size", nil, "", http.StatusInternalServerError)
	}

	if t.CreateAt == 0 {
		return NewAppError("Token.IsValid", "model.token.is_valid.expiry", nil, "", http.StatusInternalServerError)
	}
	return nil
}
