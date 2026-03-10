package draft

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Draft struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DraftStore struct {
	filePath string
	drafts   map[string]Draft
}

func NewDraftStore() (*DraftStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	draftsPath := filepath.Join(home, ".config", "x-cli", "drafts.json")
	store := &DraftStore{
		filePath: draftsPath,
		drafts:   make(map[string]Draft),
	}

	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

func (s *DraftStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.drafts)
}

func (s *DraftStore) save() error {
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.drafts, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func (s *DraftStore) Save(text string) (*Draft, error) {
	id := generateID()
	now := time.Now()

	draft := Draft{
		ID:        id,
		Text:      text,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.drafts[id] = draft

	if err := s.save(); err != nil {
		return nil, err
	}

	return &draft, nil
}

func (s *DraftStore) Update(id string, text string) (*Draft, error) {
	draft, exists := s.drafts[id]
	if !exists {
		return nil, fmt.Errorf("draft not found: %s", id)
	}

	draft.Text = text
	draft.UpdatedAt = time.Now()
	s.drafts[id] = draft

	if err := s.save(); err != nil {
		return nil, err
	}

	return &draft, nil
}

func (s *DraftStore) Get(id string) (*Draft, error) {
	draft, exists := s.drafts[id]
	if !exists {
		return nil, fmt.Errorf("draft not found: %s", id)
	}

	return &draft, nil
}

func (s *DraftStore) Delete(id string) error {
	if _, exists := s.drafts[id]; !exists {
		return fmt.Errorf("draft not found: %s", id)
	}

	delete(s.drafts, id)

	return s.save()
}

func (s *DraftStore) List() []Draft {
	drafts := make([]Draft, 0, len(s.drafts))
	for _, draft := range s.drafts {
		drafts = append(drafts, draft)
	}

	sort.Slice(drafts, func(i, j int) bool {
		return drafts[i].UpdatedAt.After(drafts[j].UpdatedAt)
	})

	return drafts
}

func (s *DraftStore) Count() int {
	return len(s.drafts)
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
