package domain

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
