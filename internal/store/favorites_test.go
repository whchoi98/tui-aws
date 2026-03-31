// internal/store/favorites_test.go
package store

import (
	"path/filepath"
	"testing"
)

func TestFavoritesAddRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "favorites.json")
	f, err := LoadFavorites(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	fav := Favorite{
		InstanceID: "i-abc123",
		Profile:    "prod",
		Region:     "ap-northeast-2",
		Alias:      "web-server-1",
	}

	f.Add(fav)
	if len(f.Items) != 1 {
		t.Fatalf("expected 1 favorite, got %d", len(f.Items))
	}

	if !f.IsFavorite("i-abc123", "prod", "ap-northeast-2") {
		t.Error("expected i-abc123 to be a favorite")
	}

	f.Remove("i-abc123", "prod", "ap-northeast-2")
	if len(f.Items) != 0 {
		t.Fatalf("expected 0 favorites after remove, got %d", len(f.Items))
	}

	if err := f.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadFavorites(path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if len(loaded.Items) != 0 {
		t.Errorf("expected 0 favorites after reload, got %d", len(loaded.Items))
	}
}

func TestFavoritesNoDuplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "favorites.json")
	f, _ := LoadFavorites(path)

	fav := Favorite{InstanceID: "i-abc123", Profile: "prod", Region: "us-east-1", Alias: "web"}
	f.Add(fav)
	f.Add(fav)
	if len(f.Items) != 1 {
		t.Errorf("expected 1 favorite (no dup), got %d", len(f.Items))
	}
}
