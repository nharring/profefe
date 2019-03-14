package profile

import (
	"time"

	"github.com/rs/xid"
)

type Profile struct {
	Type    ProfileType
	Service *Service
}

type Token xid.ID

func TokenFromString(s string) Token {
	token, _ := xid.FromString(s)
	return Token(token)
}

func (token Token) MarshalJSON() ([]byte, error) {
	return xid.ID(token).MarshalJSON()
}

func (token Token) String() string {
	return xid.ID(token).String()
}

type Service struct {
	Name      string
	BuildID   string
	Token     Token `json:",string"`
	Labels    Labels
	CreatedAt time.Time
}

func NewService(name, buildid string, labels Labels) *Service {
	return &Service{
		Name:      name,
		BuildID:   buildid,
		Token:     Token(xid.New()),
		Labels:    labels,
		CreatedAt: time.Now().UTC(),
	}
}
