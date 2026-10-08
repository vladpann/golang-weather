package users_service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	core_errors "github.com/vladpann/golang-weather/internal/core/errors"
	users_repository_postgres "github.com/vladpann/golang-weather/internal/features/users/repository/postgres"
	users_service "github.com/vladpann/golang-weather/internal/features/users/service"
	"github.com/vladpann/golang-weather/internal/testutil"
)

func TestUsersService_CreateUser_Integration(t *testing.T) {
	// Подготовка тестовой базы данных
	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, pool)

	// Инициализация зависимостей
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	ctx := context.Background()

	t.Run("успешная регистрация нового пользователя", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Входные данные
		login := "testuser"
		password := "password123"
		repeatPassword := "password123"

		// Выполнение
		user, err := usersService.CreateUser(ctx, login, password, repeatPassword)

		// Проверка
		require.NoError(t, err)
		assert.NotZero(t, user.ID, "ID пользователя должен быть установлен")
		assert.Equal(t, login, user.Login)

		// Проверяем, что пользователь действительно создан в БД
		retrievedUser, passwordHash, err := usersRepository.GetUser(ctx, login)
		require.NoError(t, err)
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Equal(t, login, retrievedUser.Login)
		assert.NotEmpty(t, passwordHash, "Хэш пароля должен быть сохранен")

		// Проверяем, что можно авторизоваться с этим паролем
		err = passwordHasher.Compare(passwordHash, password)
		assert.NoError(t, err, "Сохраненный хэш должен соответствовать паролю")
	})

	t.Run("регистрация с неуникальным логином приводит к ошибке", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		login := "duplicateuser"
		password := "password123"
		repeatPassword := "password123"

		// Создаем первого пользователя
		user1, err := usersService.CreateUser(ctx, login, password, repeatPassword)
		require.NoError(t, err)
		assert.NotZero(t, user1.ID)

		// Пытаемся создать второго пользователя с тем же логином
		user2, err := usersService.CreateUser(ctx, login, password, repeatPassword)

		// Проверка
		require.Error(t, err)
		assert.ErrorIs(t, err, core_errors.ErrLoginAlreadyExists, "Должна быть ошибка о существующем логине")
		assert.Zero(t, user2.ID, "Второй пользователь не должен быть создан")

		// Проверяем, что в БД только один пользователь
		retrievedUser, _, err := usersRepository.GetUser(ctx, login)
		require.NoError(t, err)
		assert.Equal(t, user1.ID, retrievedUser.ID, "В БД должен быть только первый пользователь")
	})

	t.Run("регистрация с несовпадающими паролями приводит к ошибке", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		login := "testuser2"
		password := "password123"
		repeatPassword := "differentpassword"

		// Выполнение
		user, err := usersService.CreateUser(ctx, login, password, repeatPassword)

		// Проверка
		require.Error(t, err)
		assert.Contains(t, err.Error(), "password do not match", "Должна быть ошибка о несовпадении паролей")
		assert.Zero(t, user.ID, "Пользователь не должен быть создан")
	})

	t.Run("регистрация с пустым логином приводит к ошибке", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		login := ""
		password := "password123"
		repeatPassword := "password123"

		// Выполнение
		user, err := usersService.CreateUser(ctx, login, password, repeatPassword)

		// Проверка
		require.Error(t, err)
		assert.Zero(t, user.ID, "Пользователь не должен быть создан")
	})

	t.Run("регистрация с слишком коротким паролем приводит к ошибке", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		login := "testuser3"
		password := "123" // Меньше минимальной длины (4 символа)
		repeatPassword := "123"

		// Выполнение
		user, err := usersService.CreateUser(ctx, login, password, repeatPassword)

		// Проверка
		require.Error(t, err)
		assert.Zero(t, user.ID, "Пользователь не должен быть создан")
	})
}

func TestUsersService_GetUser_Integration(t *testing.T) {
	// Подготовка тестовой базы данных
	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, pool)

	// Инициализация зависимостей
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	ctx := context.Background()

	t.Run("успешная авторизация с правильным паролем", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем пользователя
		login := "authuser"
		password := "password123"
		repeatPassword := "password123"

		createdUser, err := usersService.CreateUser(ctx, login, password, repeatPassword)
		require.NoError(t, err)

		// Пытаемся получить пользователя с правильным паролем
		retrievedUser, err := usersService.GetUser(ctx, login, password)

		// Проверка
		require.NoError(t, err)
		assert.Equal(t, createdUser.ID, retrievedUser.ID)
		assert.Equal(t, createdUser.Login, retrievedUser.Login)
	})

	t.Run("авторизация с неправильным паролем приводит к ошибке", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем пользователя
		login := "authuser2"
		password := "password123"
		repeatPassword := "password123"

		_, err := usersService.CreateUser(ctx, login, password, repeatPassword)
		require.NoError(t, err)

		// Пытаемся получить пользователя с неправильным паролем
		wrongPassword := "wrongpassword"
		user, err := usersService.GetUser(ctx, login, wrongPassword)

		// Проверка
		require.Error(t, err)
		assert.Zero(t, user.ID, "Пользователь не должен быть возвращен")
	})

	t.Run("авторизация несуществующего пользователя приводит к ошибке", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Пытаемся получить несуществующего пользователя
		login := "nonexistentuser"
		password := "password123"

		user, err := usersService.GetUser(ctx, login, password)

		// Проверка
		require.Error(t, err)
		assert.Zero(t, user.ID, "Пользователь не должен быть возвращен")
	})
}
