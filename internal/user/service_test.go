package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"authe/internal/platform/security"
	"authe/internal/user"
	"authe/internal/user/mocks"
)

func TestService_Get_CacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()
	expectedUser := &user.User{
		ID:       1,
		Username: "john_doe",
		Email:    "john@example.com",
	}

	// Ожидаем, что сервис обратится к кэшу и получит пользователя
	mockCache.EXPECT().
		GetUserById(ctx, 1).
		Return(expectedUser, nil).
		Times(1)

	// В базу данных обращений быть не должно
	mockRepo.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Times(0)

	resultUser, err := svc.Get(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, expectedUser, resultUser)
}

func TestService_Get_CacheMiss_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()
	expectedUser := &user.User{
		ID:       1,
		Username: "john_doe",
		Email:    "john@example.com",
	}

	// 1. Проверяем кэш -> промах (ошибка / не найден)
	mockCache.EXPECT().
		GetUserById(ctx, 1).
		Return(nil, errors.New("cache miss")).
		Times(1)

	// 2. Идем в базу данных -> находим пользователя
	mockRepo.EXPECT().
		Get(ctx, 1).
		Return(expectedUser, nil).
		Times(1)

	// 3. Сохраняем полученного пользователя в кэш на 15 минут
	mockCache.EXPECT().
		SaveUser(ctx, expectedUser, 15*time.Minute).
		Return(nil).
		Times(1)

	resultUser, err := svc.Get(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, expectedUser, resultUser)
}

func TestService_Get_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()

	// 1. Проверяем кэш -> промах
	mockCache.EXPECT().
		GetUserById(ctx, 42).
		Return(nil, errors.New("cache miss")).
		Times(1)

	// 2. Идем в базу данных -> пользователь не найден
	mockRepo.EXPECT().
		Get(ctx, 42).
		Return(nil, user.UserNotFound).
		Times(1)

	// В кэш ничего сохраняться не должно
	mockCache.EXPECT().
		SaveUser(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	resultUser, err := svc.Get(ctx, 42)

	require.Error(t, err)
	assert.Nil(t, resultUser)
	assert.True(t, errors.Is(err, user.UserNotFound))
}

func TestService_Create_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()
	newUser := &user.User{
		Username: "newuser",
		Password: "plainpassword123",
		Email:    "newuser@example.com",
	}

	// Репозиторий должен получить пользователя с захэшированным паролем
	mockRepo.EXPECT().
		Create(ctx, gomock.Cond(func(u *user.User) bool {
			return u.Username == "newuser" &&
				u.PasswordHash != "" &&
				security.CheckPassword("plainpassword123", u.PasswordHash)
		})).
		Return(10, nil).
		Times(1)

	createdID, err := svc.Create(ctx, newUser)

	require.NoError(t, err)
	assert.Equal(t, 10, createdID)
}

func TestService_Login_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()
	password := "correct_password"
	hash, err := security.HashPassword(password)
	require.NoError(t, err)

	existingUser := &user.User{
		ID:           1,
		Username:     "alice",
		PasswordHash: hash,
	}

	// 1. Поиск пользователя по username
	mockRepo.EXPECT().
		GetByUsername(ctx, "alice").
		Return(existingUser, nil).
		Times(1)

	// 2. Создание токена
	mockTokenManager.EXPECT().
		CreateToken("alice").
		Return("jwt-token-xyz", nil).
		Times(1)

	token, err := svc.Login(ctx, "alice", password)

	require.NoError(t, err)
	assert.Equal(t, "Bearer jwt-token-xyz", token)
}

func TestService_Login_WrongPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()
	hash, err := security.HashPassword("correct_password")
	require.NoError(t, err)

	existingUser := &user.User{
		ID:           1,
		Username:     "alice",
		PasswordHash: hash,
	}

	// 1. Поиск пользователя по username
	mockRepo.EXPECT().
		GetByUsername(ctx, "alice").
		Return(existingUser, nil).
		Times(1)

	// Токен НЕ должен генерироваться
	mockTokenManager.EXPECT().
		CreateToken(gomock.Any()).
		Times(0)

	token, err := svc.Login(ctx, "alice", "wrong_password")

	require.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "invalid password")
}

func TestService_Login_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()

	mockRepo.EXPECT().
		GetByUsername(ctx, "unknown_user").
		Return(nil, user.UserNotFound).
		Times(1)

	token, err := svc.Login(ctx, "unknown_user", "password")

	require.Error(t, err)
	assert.Empty(t, token)
	assert.True(t, errors.Is(err, user.UserNotFound))
}

func TestService_Logout_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()
	rawToken := "valid-token-string"
	bearerToken := "Bearer " + rawToken

	expTime := time.Now().Add(10 * time.Minute)
	claims := jwt.MapClaims{
		"username": "alice",
		"exp":      float64(expTime.Unix()),
	}

	// 1. Извлечение claims
	mockTokenManager.EXPECT().
		ExtractClaimsWithMap(rawToken).
		Return(claims, nil).
		Times(1)

	// 2. Добавление в черный список
	mockCache.EXPECT().
		BlacklistToken(ctx, rawToken, gomock.Cond(func(d time.Duration) bool {
			return d > 0 && d <= 10*time.Minute
		})).
		Return(nil).
		Times(1)

	err := svc.Logout(ctx, bearerToken)

	require.NoError(t, err)
}

func TestService_Logout_InvalidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()
	rawToken := "invalid-token"

	mockTokenManager.EXPECT().
		ExtractClaimsWithMap(rawToken).
		Return(nil, errors.New("invalid signature")).
		Times(1)

	mockCache.EXPECT().
		BlacklistToken(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	err := svc.Logout(ctx, rawToken)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract claims")
}

func TestService_Update_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()
	updateUser := &user.User{
		Password: "newpassword123",
		Email:    "new@example.com",
	}

	mockRepo.EXPECT().
		Update(ctx, 1, gomock.Cond(func(u *user.User) bool {
			return u.PasswordHash != "" && security.CheckPassword("newpassword123", u.PasswordHash)
		})).
		Return(nil).
		Times(1)

	mockCache.EXPECT().
		DeleteUser(ctx, 1).
		Return(nil).
		Times(1)

	err := svc.Update(ctx, 1, updateUser)
	require.NoError(t, err)
}

func TestService_Delete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()

	mockRepo.EXPECT().
		Delete(ctx, 1).
		Return(nil).
		Times(1)

	mockCache.EXPECT().
		DeleteUser(ctx, 1).
		Return(nil).
		Times(1)

	err := svc.Delete(ctx, 1)
	require.NoError(t, err)
}

func TestService_IsExistByUsername(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockRepository(ctrl)
	mockCache := mocks.NewMockCache(ctrl)
	mockTokenManager := mocks.NewMockTokenManager(ctrl)

	svc := user.NewService(mockRepo, mockCache, mockTokenManager)

	ctx := context.Background()

	t.Run("user exists", func(t *testing.T) {
		mockRepo.EXPECT().
			GetByUsername(ctx, "existing_user").
			Return(&user.User{ID: 1, Username: "existing_user"}, nil).
			Times(1)

		err := svc.IsExistByUsername(ctx, "existing_user")
		require.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		mockRepo.EXPECT().
			GetByUsername(ctx, "not_found").
			Return(nil, user.UserNotFound).
			Times(1)

		err := svc.IsExistByUsername(ctx, "not_found")
		require.Error(t, err)
		assert.True(t, errors.Is(err, user.UserNotFound))
	})
}
