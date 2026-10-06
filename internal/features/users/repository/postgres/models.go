package users_repository_postgres

import "github.com/vladpann/golang-weather/internal/core/domain"

type UserModel struct {
	ID    int64
	Login string
}

func userDomainFromModel(user UserModel) domain.User {
	return domain.NewUser(
		user.ID,
		user.Login,
	)
}
