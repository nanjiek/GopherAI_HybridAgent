package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gophermind/internal/config"
	"gophermind/internal/core/model"
	"gophermind/internal/security/token"
)

type fakeAuthRepo struct {
	users    map[string]model.AuthUser
	refresh  map[string]model.RefreshTokenRecord
	notFound error
}

func newFakeAuthRepo() *fakeAuthRepo {
	return &fakeAuthRepo{
		users:    map[string]model.AuthUser{},
		refresh:  map[string]model.RefreshTokenRecord{},
		notFound: errors.New("not found"),
	}
}

func (f *fakeAuthRepo) CreateUser(_ context.Context, username string, passwordHash string, role string) (model.AuthUser, error) {
	if _, ok := f.users[username]; ok {
		return model.AuthUser{}, errors.New("duplicate")
	}
	u := model.AuthUser{
		ID:           uint64(len(f.users) + 1),
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
	}
	f.users[username] = u
	return u, nil
}

func (f *fakeAuthRepo) GetUserByUsername(_ context.Context, username string) (model.AuthUser, error) {
	u, ok := f.users[username]
	if !ok {
		return model.AuthUser{}, f.notFound
	}
	return u, nil
}

func (f *fakeAuthRepo) SaveRefreshToken(_ context.Context, userID uint64, tokenJTI string, tokenHash string, deviceID string, expiresAt time.Time) error {
	f.refresh[tokenJTI] = model.RefreshTokenRecord{
		UserID:    userID,
		TokenJTI:  tokenJTI,
		TokenHash: tokenHash,
		DeviceID:  deviceID,
		ExpiresAt: expiresAt,
	}
	return nil
}

func (f *fakeAuthRepo) GetActiveRefreshToken(_ context.Context, tokenJTI string, tokenHash string) (model.RefreshTokenRecord, error) {
	r, ok := f.refresh[tokenJTI]
	if !ok || r.TokenHash != tokenHash {
		return model.RefreshTokenRecord{}, f.notFound
	}
	return r, nil
}

func (f *fakeAuthRepo) RevokeRefreshToken(_ context.Context, tokenJTI string) error {
	delete(f.refresh, tokenJTI)
	return nil
}

func (f *fakeAuthRepo) RevokeAllRefreshTokens(_ context.Context, userID uint64) error {
	for k, v := range f.refresh {
		if v.UserID == userID {
			delete(f.refresh, k)
		}
	}
	return nil
}

func (f *fakeAuthRepo) IsNotFound(err error) bool {
	return errors.Is(err, f.notFound)
}

func TestAuthService_LoginAndRefresh(t *testing.T) {
	repo := newFakeAuthRepo()
	mgr := token.NewManager(config.AuthConfig{
		AccessSecret:  "access-secret",
		RefreshSecret: "refresh-secret",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    24 * time.Hour,
	})
	svc := NewAuthService(repo, mgr)

	require.NoError(t, svc.Register(context.Background(), "alice", "password123"))
	pair, err := svc.Login(context.Background(), "alice", "password123", "device-a")
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)

	newPair, err := svc.Refresh(context.Background(), pair.RefreshToken, "device-a")
	require.NoError(t, err)
	require.NotEqual(t, pair.RefreshToken, newPair.RefreshToken)
}
