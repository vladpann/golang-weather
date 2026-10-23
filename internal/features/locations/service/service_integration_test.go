package locations_service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
	locations_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/locations/repository/postgres"
	locations_service "github.com/vladpann/backend/golang-weather/internal/features/locations/service"
	users_repository_postgres "github.com/vladpann/backend/golang-weather/internal/features/users/repository/postgres"
	users_service "github.com/vladpann/backend/golang-weather/internal/features/users/service"
	"github.com/vladpann/backend/golang-weather/internal/testutil"
)

func TestLocationsService_CreateLocation_Integration(t *testing.T) {
	// Подготовка тестовой базы данных
	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, pool)

	// Инициализация зависимостей
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	locationsRepository := locations_repository_postgres.NewLocationsRepository(pool)
	locationsService := locations_service.NewLocationsService(locationsRepository)

	ctx := context.Background()

	t.Run("успешное создание локации для пользователя", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем пользователя
		user, err := usersService.CreateUser(ctx, "locationuser", "password123", "password123")
		require.NoError(t, err)

		// Создаем локацию
		location := domain.NewLocation(
			0, // ID будет установлен БД
			user.ID,
			"Moscow",
			55.7558,
			37.6173,
		)

		createdLocation, err := locationsService.CreateLocation(ctx, location)

		// Проверка
		require.NoError(t, err)
		assert.NotZero(t, createdLocation.ID, "ID локации должен быть установлен")
		assert.Equal(t, user.ID, createdLocation.UserID)
		assert.Equal(t, "Moscow", createdLocation.Name)
		assert.Equal(t, 55.7558, createdLocation.Latitude)
		assert.Equal(t, 37.6173, createdLocation.Longitude)

		// Проверяем, что локация действительно создана в БД
		retrievedLocation, err := locationsRepository.GetLocation(ctx, createdLocation.ID, user.ID)
		require.NoError(t, err)
		assert.Equal(t, createdLocation.ID, retrievedLocation.ID)
		assert.Equal(t, createdLocation.Name, retrievedLocation.Name)
	})

	t.Run("создание дубликата локации для одного пользователя приводит к ошибке", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем пользователя
		user, err := usersService.CreateUser(ctx, "duplicateuser", "password123", "password123")
		require.NoError(t, err)

		// Создаем первую локацию
		location1 := domain.NewLocation(0, user.ID, "London", 51.5074, -0.1278)
		createdLocation1, err := locationsService.CreateLocation(ctx, location1)
		require.NoError(t, err)
		assert.NotZero(t, createdLocation1.ID)

		// Пытаемся создать дубликат локации с теми же координатами
		location2 := domain.NewLocation(0, user.ID, "London City", 51.5074, -0.1278)
		_, err = locationsService.CreateLocation(ctx, location2)

		// Проверка
		require.Error(t, err, "Создание дубликата должно привести к ошибке")
		assert.ErrorIs(t, err, core_errors.ErrLocationAlreadyExists, "Должна быть ошибка о существующей локации")

		// Проверяем, что в БД только одна локация
		locations, err := locationsRepository.GetLocations(ctx, user.ID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, locations, 1, "В БД должна быть только одна локация")
		assert.Equal(t, createdLocation1.ID, locations[0].ID)
	})

	t.Run("создание одинаковых локаций разными пользователями должно работать", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем двух пользователей
		user1, err := usersService.CreateUser(ctx, "user1", "password123", "password123")
		require.NoError(t, err)

		user2, err := usersService.CreateUser(ctx, "user2", "password123", "password123")
		require.NoError(t, err)

		// Оба пользователя добавляют локацию с одинаковыми координатами
		location1 := domain.NewLocation(0, user1.ID, "Paris", 48.8566, 2.3522)
		createdLocation1, err := locationsService.CreateLocation(ctx, location1)
		require.NoError(t, err)

		location2 := domain.NewLocation(0, user2.ID, "Paris", 48.8566, 2.3522)
		createdLocation2, err := locationsService.CreateLocation(ctx, location2)

		// Проверка
		require.NoError(t, err, "Разные пользователи могут добавлять одинаковые локации")
		assert.NotEqual(t, createdLocation1.ID, createdLocation2.ID, "Локации должны иметь разные ID")
		assert.Equal(t, user1.ID, createdLocation1.UserID)
		assert.Equal(t, user2.ID, createdLocation2.UserID)
	})

	t.Run("создание локации с экстремальными координатами", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		user, err := usersService.CreateUser(ctx, "extremeuser", "password123", "password123")
		require.NoError(t, err)

		testCases := []struct {
			name      string
			cityName  string
			latitude  float64
			longitude float64
		}{
			{"северный полюс", "North Pole", 90.0, 0.0},
			{"южный полюс", "South Pole", -90.0, 0.0},
			{"линия перемены дат восток", "Date Line East", 0.0, 180.0},
			{"линия перемены дат запад", "Date Line West", 0.0, -180.0},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				location := domain.NewLocation(0, user.ID, tc.cityName, tc.latitude, tc.longitude)
				createdLocation, err := locationsService.CreateLocation(ctx, location)

				require.NoError(t, err)
				assert.Equal(t, tc.latitude, createdLocation.Latitude)
				assert.Equal(t, tc.longitude, createdLocation.Longitude)
			})
		}
	})
}

func TestLocationsService_GetLocation_Integration(t *testing.T) {
	// Подготовка тестовой базы данных
	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, pool)

	// Инициализация зависимостей
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	locationsRepository := locations_repository_postgres.NewLocationsRepository(pool)
	locationsService := locations_service.NewLocationsService(locationsRepository)

	ctx := context.Background()

	t.Run("успешное получение локации владельцем", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем пользователя и локацию
		user, err := usersService.CreateUser(ctx, "owner", "password123", "password123")
		require.NoError(t, err)

		location := domain.NewLocation(0, user.ID, "Berlin", 52.5200, 13.4050)
		createdLocation, err := locationsService.CreateLocation(ctx, location)
		require.NoError(t, err)

		// Получаем локацию
		retrievedLocation, err := locationsService.GetLocation(ctx, createdLocation.ID, user.ID)

		// Проверка
		require.NoError(t, err)
		assert.Equal(t, createdLocation.ID, retrievedLocation.ID)
		assert.Equal(t, createdLocation.UserID, retrievedLocation.UserID)
		assert.Equal(t, createdLocation.Name, retrievedLocation.Name)
		assert.Equal(t, createdLocation.Latitude, retrievedLocation.Latitude)
		assert.Equal(t, createdLocation.Longitude, retrievedLocation.Longitude)
	})

	t.Run("попытка получить чужую локацию возвращает ошибку", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем двух пользователей
		user1, err := usersService.CreateUser(ctx, "user1", "password123", "password123")
		require.NoError(t, err)

		user2, err := usersService.CreateUser(ctx, "user2", "password123", "password123")
		require.NoError(t, err)

		// user1 создает локацию
		location := domain.NewLocation(0, user1.ID, "Tokyo", 35.6762, 139.6503)
		createdLocation, err := locationsService.CreateLocation(ctx, location)
		require.NoError(t, err)

		// user2 пытается получить локацию user1
		_, err = locationsService.GetLocation(ctx, createdLocation.ID, user2.ID)

		// Проверка
		require.Error(t, err, "Попытка получить чужую локацию должна вернуть ошибку")
		assert.ErrorIs(t, err, core_errors.ErrNotFound)
	})

	t.Run("попытка получить несуществующую локацию возвращает ошибку", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		user, err := usersService.CreateUser(ctx, "user", "password123", "password123")
		require.NoError(t, err)

		// Пытаемся получить несуществующую локацию
		nonExistentID := int64(99999)
		_, err = locationsService.GetLocation(ctx, nonExistentID, user.ID)

		// Проверка
		require.Error(t, err)
		assert.ErrorIs(t, err, core_errors.ErrNotFound)
	})
}

func TestLocationsService_GetLocations_Integration(t *testing.T) {
	// Подготовка тестовой базы данных
	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, pool)

	// Инициализация зависимостей
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	locationsRepository := locations_repository_postgres.NewLocationsRepository(pool)
	locationsService := locations_service.NewLocationsService(locationsRepository)

	ctx := context.Background()

	t.Run("получение списка локаций для пользователя с несколькими локациями", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем пользователя
		user, err := usersService.CreateUser(ctx, "multilocuser", "password123", "password123")
		require.NoError(t, err)

		// Создаем несколько локаций
		locations := []domain.Location{
			domain.NewLocation(0, user.ID, "New York", 40.7128, -74.0060),
			domain.NewLocation(0, user.ID, "Los Angeles", 34.0522, -118.2437),
			domain.NewLocation(0, user.ID, "Chicago", 41.8781, -87.6298),
		}

		var createdIDs []int64
		for _, loc := range locations {
			created, err := locationsService.CreateLocation(ctx, loc)
			require.NoError(t, err)
			createdIDs = append(createdIDs, created.ID)
		}

		// Получаем список локаций
		retrievedLocations, err := locationsService.GetLocations(ctx, user.ID, nil, nil)

		// Проверка
		require.NoError(t, err)
		assert.Len(t, retrievedLocations, 3, "Должны быть получены все 3 локации")

		// Проверяем, что все созданные локации присутствуют
		for _, createdID := range createdIDs {
			found := false
			for _, retrieved := range retrievedLocations {
				if retrieved.ID == createdID {
					found = true
					break
				}
			}
			assert.True(t, found, "Локация с ID %d должна быть в списке", createdID)
		}
	})

	t.Run("получение пустого списка для нового пользователя", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем пользователя без локаций
		user, err := usersService.CreateUser(ctx, "newuser", "password123", "password123")
		require.NoError(t, err)

		// Получаем список локаций
		locations, err := locationsService.GetLocations(ctx, user.ID, nil, nil)

		// Проверка
		require.NoError(t, err)
		assert.Empty(t, locations, "Список локаций для нового пользователя должен быть пустым")
	})

	t.Run("пользователь получает только свои локации", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем двух пользователей
		user1, err := usersService.CreateUser(ctx, "user1", "password123", "password123")
		require.NoError(t, err)

		user2, err := usersService.CreateUser(ctx, "user2", "password123", "password123")
		require.NoError(t, err)

		// user1 создает локации
		loc1 := domain.NewLocation(0, user1.ID, "Madrid", 40.4168, -3.7038)
		created1, err := locationsService.CreateLocation(ctx, loc1)
		require.NoError(t, err)

		loc2 := domain.NewLocation(0, user1.ID, "Barcelona", 41.3851, 2.1734)
		created2, err := locationsService.CreateLocation(ctx, loc2)
		require.NoError(t, err)

		// user2 создает локацию
		loc3 := domain.NewLocation(0, user2.ID, "Rome", 41.9028, 12.4964)
		_, err = locationsService.CreateLocation(ctx, loc3)
		require.NoError(t, err)

		// user1 получает свои локации
		user1Locations, err := locationsService.GetLocations(ctx, user1.ID, nil, nil)

		// Проверка
		require.NoError(t, err)
		assert.Len(t, user1Locations, 2, "user1 должен видеть только свои локации")

		// Проверяем, что локации принадлежат user1
		for _, loc := range user1Locations {
			assert.Equal(t, user1.ID, loc.UserID)
		}

		// Проверяем ID локаций
		assert.Contains(t, []int64{user1Locations[0].ID, user1Locations[1].ID}, created1.ID)
		assert.Contains(t, []int64{user1Locations[0].ID, user1Locations[1].ID}, created2.ID)
	})

	t.Run("получение локаций с limit", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		user, err := usersService.CreateUser(ctx, "limituser", "password123", "password123")
		require.NoError(t, err)

		// Создаем 5 локаций
		for i := 0; i < 5; i++ {
			loc := domain.NewLocation(0, user.ID, "City", float64(i), float64(i))
			_, err := locationsService.CreateLocation(ctx, loc)
			require.NoError(t, err)
		}

		// Получаем только 2 локации
		limit := 2
		locations, err := locationsService.GetLocations(ctx, user.ID, &limit, nil)

		// Проверка
		require.NoError(t, err)
		assert.Len(t, locations, 2, "Должно быть получено ровно 2 локации")
	})

	t.Run("получение локаций с offset", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		user, err := usersService.CreateUser(ctx, "offsetuser", "password123", "password123")
		require.NoError(t, err)

		// Создаем 5 локаций
		var createdIDs []int64
		for i := 0; i < 5; i++ {
			loc := domain.NewLocation(0, user.ID, "City", float64(i), float64(i))
			created, err := locationsService.CreateLocation(ctx, loc)
			require.NoError(t, err)
			createdIDs = append(createdIDs, created.ID)
		}

		// Пропускаем первые 2 локации
		offset := 2
		locations, err := locationsService.GetLocations(ctx, user.ID, nil, &offset)

		// Проверка
		require.NoError(t, err)
		assert.Len(t, locations, 3, "Должно быть получено 3 локации (5 - 2)")

		// Проверяем, что первые две локации не включены
		for _, loc := range locations {
			assert.NotEqual(t, createdIDs[0], loc.ID)
			assert.NotEqual(t, createdIDs[1], loc.ID)
		}
	})

	t.Run("ошибка при негативном limit", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		user, err := usersService.CreateUser(ctx, "user", "password123", "password123")
		require.NoError(t, err)

		// Пытаемся использовать негативный limit
		negativeLimit := -1
		_, err = locationsService.GetLocations(ctx, user.ID, &negativeLimit, nil)

		// Проверка
		require.Error(t, err)
		assert.ErrorIs(t, err, core_errors.ErrInvalidArgument)
	})

	t.Run("ошибка при негативном offset", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		user, err := usersService.CreateUser(ctx, "user", "password123", "password123")
		require.NoError(t, err)

		// Пытаемся использовать негативный offset
		negativeOffset := -1
		_, err = locationsService.GetLocations(ctx, user.ID, nil, &negativeOffset)

		// Проверка
		require.Error(t, err)
		assert.ErrorIs(t, err, core_errors.ErrInvalidArgument)
	})
}

func TestLocationsService_DeleteLocation_Integration(t *testing.T) {
	// Подготовка тестовой базы данных
	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, pool)

	// Инициализация зависимостей
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	locationsRepository := locations_repository_postgres.NewLocationsRepository(pool)
	locationsService := locations_service.NewLocationsService(locationsRepository)

	ctx := context.Background()

	t.Run("успешное удаление собственной локации", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем пользователя и локацию
		user, err := usersService.CreateUser(ctx, "deleteuser", "password123", "password123")
		require.NoError(t, err)

		location := domain.NewLocation(0, user.ID, "Vienna", 48.2082, 16.3738)
		createdLocation, err := locationsService.CreateLocation(ctx, location)
		require.NoError(t, err)

		// Удаляем локацию
		err = locationsService.DeleteLocation(ctx, createdLocation.ID)

		// Проверка
		require.NoError(t, err)

		// Проверяем, что локация действительно удалена
		_, err = locationsRepository.GetLocation(ctx, createdLocation.ID, user.ID)
		require.Error(t, err, "Локация должна быть удалена")
		assert.ErrorIs(t, err, core_errors.ErrNotFound)

		// Проверяем, что список локаций пуст
		locations, err := locationsRepository.GetLocations(ctx, user.ID, nil, nil)
		require.NoError(t, err)
		assert.Empty(t, locations, "Список локаций должен быть пустым после удаления")
	})

	t.Run("попытка удалить несуществующую локацию возвращает ошибку", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Пытаемся удалить несуществующую локацию
		nonExistentID := int64(99999)
		err := locationsService.DeleteLocation(ctx, nonExistentID)

		// Проверка
		require.Error(t, err)
		assert.ErrorIs(t, err, core_errors.ErrNotFound)
	})

	t.Run("удаление не влияет на локации других пользователей", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем двух пользователей с локациями
		user1, err := usersService.CreateUser(ctx, "user1", "password123", "password123")
		require.NoError(t, err)

		user2, err := usersService.CreateUser(ctx, "user2", "password123", "password123")
		require.NoError(t, err)

		loc1 := domain.NewLocation(0, user1.ID, "Amsterdam", 52.3676, 4.9041)
		created1, err := locationsService.CreateLocation(ctx, loc1)
		require.NoError(t, err)

		loc2 := domain.NewLocation(0, user2.ID, "Brussels", 50.8503, 4.3517)
		created2, err := locationsService.CreateLocation(ctx, loc2)
		require.NoError(t, err)

		// user1 удаляет свою локацию
		err = locationsService.DeleteLocation(ctx, created1.ID)
		require.NoError(t, err)

		// Проверяем, что локация user2 осталась
		retrievedLocation, err := locationsRepository.GetLocation(ctx, created2.ID, user2.ID)
		require.NoError(t, err)
		assert.Equal(t, created2.ID, retrievedLocation.ID)
		assert.Equal(t, user2.ID, retrievedLocation.UserID)

		// Проверяем, что у user1 нет локаций
		user1Locations, err := locationsRepository.GetLocations(ctx, user1.ID, nil, nil)
		require.NoError(t, err)
		assert.Empty(t, user1Locations)

		// Проверяем, что у user2 есть локация
		user2Locations, err := locationsRepository.GetLocations(ctx, user2.ID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, user2Locations, 1)
	})

	t.Run("удаление одной локации из нескольких", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// Создаем пользователя с несколькими локациями
		user, err := usersService.CreateUser(ctx, "multiuser", "password123", "password123")
		require.NoError(t, err)

		loc1 := domain.NewLocation(0, user.ID, "City1", 10.0, 20.0)
		created1, err := locationsService.CreateLocation(ctx, loc1)
		require.NoError(t, err)

		loc2 := domain.NewLocation(0, user.ID, "City2", 30.0, 40.0)
		created2, err := locationsService.CreateLocation(ctx, loc2)
		require.NoError(t, err)

		loc3 := domain.NewLocation(0, user.ID, "City3", 50.0, 60.0)
		created3, err := locationsService.CreateLocation(ctx, loc3)
		require.NoError(t, err)

		// Удаляем средню локацию
		err = locationsService.DeleteLocation(ctx, created2.ID)
		require.NoError(t, err)

		// Проверяем, что осталось 2 локации
		locations, err := locationsRepository.GetLocations(ctx, user.ID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, locations, 2)

		// Проверяем, что остались правильные локации
		ids := []int64{locations[0].ID, locations[1].ID}
		assert.Contains(t, ids, created1.ID)
		assert.Contains(t, ids, created3.ID)
		assert.NotContains(t, ids, created2.ID)
	})
}

func TestLocationsService_CompleteScenario_Integration(t *testing.T) {
	// Подготовка тестовой базы данных
	pool := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, pool)

	// Инициализация зависимостей
	usersRepository := users_repository_postgres.NewUsersRepository(pool)
	passwordHasher := users_service.NewPasswordHasher()
	usersService := users_service.NewUsersService(usersRepository, passwordHasher)

	locationsRepository := locations_repository_postgres.NewLocationsRepository(pool)
	locationsService := locations_service.NewLocationsService(locationsRepository)

	ctx := context.Background()

	t.Run("полный сценарий работы с локациями", func(t *testing.T) {
		testutil.ClearTables(t, pool)

		// 1. Регистрируем пользователя
		user, err := usersService.CreateUser(ctx, "scenariouser", "password123", "password123")
		require.NoError(t, err)
		t.Logf("Пользователь создан: ID=%d, Login=%s", user.ID, user.Login)

		// 2. Проверяем, что у пользователя пока нет локаций
		locations, err := locationsService.GetLocations(ctx, user.ID, nil, nil)
		require.NoError(t, err)
		assert.Empty(t, locations, "У нового пользователя не должно быть локаций")

		// 3. Добавляем первую локацию
		location1 := domain.NewLocation(0, user.ID, "Saint Petersburg", 59.9343, 30.3351)
		created1, err := locationsService.CreateLocation(ctx, location1)
		require.NoError(t, err)
		t.Logf("Локация 1 создана: ID=%d, Name=%s", created1.ID, created1.Name)

		// 4. Добавляем вторую локацию
		location2 := domain.NewLocation(0, user.ID, "Kazan", 55.7964, 49.1089)
		created2, err := locationsService.CreateLocation(ctx, location2)
		require.NoError(t, err)
		t.Logf("Локация 2 создана: ID=%d, Name=%s", created2.ID, created2.Name)

		// 5. Получаем список всех локаций
		locations, err = locationsService.GetLocations(ctx, user.ID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, locations, 2, "Должно быть 2 локации")

		// 6. Получаем конкретную локацию
		retrievedLocation, err := locationsService.GetLocation(ctx, created1.ID, user.ID)
		require.NoError(t, err)
		assert.Equal(t, created1.ID, retrievedLocation.ID)
		assert.Equal(t, "Saint Petersburg", retrievedLocation.Name)

		// 7. Удаляем первую локацию
		err = locationsService.DeleteLocation(ctx, created1.ID)
		require.NoError(t, err)
		t.Logf("Локация 1 удалена")

		// 8. Проверяем, что осталась только вторая локация
		locations, err = locationsService.GetLocations(ctx, user.ID, nil, nil)
		require.NoError(t, err)
		assert.Len(t, locations, 1, "Должна остаться 1 локация")
		assert.Equal(t, created2.ID, locations[0].ID)
		assert.Equal(t, "Kazan", locations[0].Name)

		// 9. Пытаемся получить удаленную локацию
		_, err = locationsService.GetLocation(ctx, created1.ID, user.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, core_errors.ErrNotFound, "Удаленная локация не должна быть найдена")
	})
}
