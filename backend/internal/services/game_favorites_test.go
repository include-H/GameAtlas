package services

import (
	"errors"
	"testing"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestGameFavoriteServiceSetBlocksAnonymousPrivateGame(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	publicID := insertServicesTestGame(t, db, "fav-public", "Fav Public", "public")
	privateID := insertServicesTestGame(t, db, "fav-private", "Fav Private", "private")

	gamesRepo := repositories.NewGamesRepository(db)
	service := NewGameFavoriteService(gamesRepo, repositories.NewFavoriteGamesRepository(db))

	// 匿名（includeAll=false）收藏公开游戏：允许
	got, err := service.Set(publicID, true, false)
	if err != nil {
		t.Fatalf("Set public game returned error: %v", err)
	}
	if !got {
		t.Fatalf("Set public game is_favorite = false, want true")
	}

	// 匿名收藏私有游戏：ErrNotFound，与"不存在"不可区分
	_, err = service.Set(privateID, true, false)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Set private game error = %v, want ErrNotFound", err)
	}

	// 管理员（includeAll=true）收藏私有游戏：允许
	got, err = service.Set(privateID, true, true)
	if err != nil {
		t.Fatalf("Set private game as admin returned error: %v", err)
	}
	if !got {
		t.Fatalf("Set private game as admin is_favorite = false, want true")
	}
}
