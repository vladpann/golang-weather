package domain

import (
	"regexp"
)

var loginRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

type User struct {
	ID    int64
	Login string
}

func NewUser(
	id int64,
	login string,
) User {
	return User{
		ID:    id,
		Login: login,
	}
}
