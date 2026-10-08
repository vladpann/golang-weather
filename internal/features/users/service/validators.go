package users_service

import "fmt"

type CreateUserInput struct {
	Login          string
	Password       string
	RepeatPassword string
}

func (i *CreateUserInput) CreateUserValidate() error {
	if i.Login == "" {
		return fmt.Errorf("login is required")
	}

	if i.Password == "" {
		return fmt.Errorf("password is required")
	}

	if len(i.Password) < 4 || len(i.Password) > 72 {
		return fmt.Errorf("password must be between 4 and 72 characters")
	}

	if i.Password != i.RepeatPassword {
		return fmt.Errorf("passwords do not match")
	}

	return nil
}
