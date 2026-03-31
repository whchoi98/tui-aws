// internal/store/favorites.go
package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Favorite struct {
	InstanceID string    `json:"instance_id"`
	Profile    string    `json:"profile"`
	Region     string    `json:"region"`
	Alias      string    `json:"alias"`
	AddedAt    time.Time `json:"added_at"`
}

type Favorites struct {
	Items []Favorite `json:"favorites"`
}

func LoadFavorites(path string) (*Favorites, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Favorites{}, nil
		}
		return nil, err
	}
	var f Favorites
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (f *Favorites) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (f *Favorites) Add(fav Favorite) {
	if f.IsFavorite(fav.InstanceID, fav.Profile, fav.Region) {
		return
	}
	fav.AddedAt = time.Now()
	f.Items = append(f.Items, fav)
}

func (f *Favorites) Remove(instanceID, profile, region string) {
	items := make([]Favorite, 0, len(f.Items))
	for _, item := range f.Items {
		if item.InstanceID == instanceID && item.Profile == profile && item.Region == region {
			continue
		}
		items = append(items, item)
	}
	f.Items = items
}

func (f *Favorites) IsFavorite(instanceID, profile, region string) bool {
	for _, item := range f.Items {
		if item.InstanceID == instanceID && item.Profile == profile && item.Region == region {
			return true
		}
	}
	return false
}

func FavoritesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tui-aws", "favorites.json")
}
