package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"
)

type Template struct {
	Name        string            `json:"name"`
	Content     string            `json:"content"`
	Category    string            `json:"category,omitempty"`
	Description string            `json:"description,omitempty"`
	Variables   map[string]string `json:"variables,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type TemplateStore struct {
	dir       string
	templates map[string]*Template
}

type Variables struct {
	Date        string
	Time        string
	DateTime    string
	Year        string
	Month       string
	Day         string
	Weekday     string
	Hour        string
	Minute      string
	Second      string
	Timestamp   string
	ISODate     string
	ISODateTime string
	Unix        string
}

func NewTemplateStore() (*TemplateStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	dir := filepath.Join(home, ".config", "x-cli", "templates")
	store := &TemplateStore{
		dir:       dir,
		templates: make(map[string]*Template),
	}

	if err := store.loadAll(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

func NewTemplateStoreWithDir(dir string) (*TemplateStore, error) {
	store := &TemplateStore{
		dir:       dir,
		templates: make(map[string]*Template),
	}

	if err := store.loadAll(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

func (s *TemplateStore) loadAll() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		filePath := filepath.Join(s.dir, entry.Name())

		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var t Template
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}

		s.templates[name] = &t
	}

	return nil
}

func (s *TemplateStore) save(t *Template) error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(s.dir, t.Name+".json")
	return os.WriteFile(filePath, data, 0644)
}

func (s *TemplateStore) Save(name, content, category, description string, variables map[string]string) (*Template, error) {
	now := time.Now()

	t := &Template{
		Name:        name,
		Content:     content,
		Category:    category,
		Description: description,
		Variables:   variables,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.save(t); err != nil {
		return nil, err
	}

	s.templates[name] = t
	return t, nil
}

func (s *TemplateStore) Update(name, content, category, description string, variables map[string]string) (*Template, error) {
	t, exists := s.templates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	t.Content = content
	if category != "" {
		t.Category = category
	}
	if description != "" {
		t.Description = description
	}
	if variables != nil {
		t.Variables = variables
	}
	t.UpdatedAt = time.Now()

	if err := s.save(t); err != nil {
		return nil, err
	}

	return t, nil
}

func (s *TemplateStore) Get(name string) (*Template, error) {
	t, exists := s.templates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	return t, nil
}

func (s *TemplateStore) Delete(name string) error {
	if _, exists := s.templates[name]; !exists {
		return fmt.Errorf("template not found: %s", name)
	}

	filePath := filepath.Join(s.dir, name+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	delete(s.templates, name)
	return nil
}

func (s *TemplateStore) List() []*Template {
	templates := make([]*Template, 0, len(s.templates))
	for _, t := range s.templates {
		templates = append(templates, t)
	}

	sort.Slice(templates, func(i, j int) bool {
		if templates[i].Category != templates[j].Category {
			return templates[i].Category < templates[j].Category
		}
		return templates[i].Name < templates[j].Name
	})

	return templates
}

func (s *TemplateStore) ListByCategory(category string) []*Template {
	templates := make([]*Template, 0)
	for _, t := range s.templates {
		if t.Category == category {
			templates = append(templates, t)
		}
	}

	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})

	return templates
}

func (s *TemplateStore) Categories() []string {
	categories := make(map[string]bool)
	for _, t := range s.templates {
		if t.Category != "" {
			categories[t.Category] = true
		}
	}

	result := make([]string, 0, len(categories))
	for cat := range categories {
		result = append(result, cat)
	}

	sort.Strings(result)
	return result
}

func (s *TemplateStore) Count() int {
	return len(s.templates)
}

func (t *Template) Render(customVars map[string]string) (string, error) {
	now := time.Now()
	vars := Variables{
		Date:        now.Format("2006-01-02"),
		Time:        now.Format("15:04:05"),
		DateTime:    now.Format("2006-01-02 15:04:05"),
		Year:        now.Format("2006"),
		Month:       now.Format("01"),
		Day:         now.Format("02"),
		Weekday:     now.Weekday().String(),
		Hour:        now.Format("15"),
		Minute:      now.Format("04"),
		Second:      now.Format("05"),
		Timestamp:   fmt.Sprintf("%d", now.Unix()),
		ISODate:     now.Format("2006-01-02"),
		ISODateTime: now.Format(time.RFC3339),
		Unix:        fmt.Sprintf("%d", now.Unix()),
	}

	data := make(map[string]interface{})
	data["Date"] = vars.Date
	data["Time"] = vars.Time
	data["DateTime"] = vars.DateTime
	data["Year"] = vars.Year
	data["Month"] = vars.Month
	data["Day"] = vars.Day
	data["Weekday"] = vars.Weekday
	data["Hour"] = vars.Hour
	data["Minute"] = vars.Minute
	data["Second"] = vars.Second
	data["Timestamp"] = vars.Timestamp
	data["ISODate"] = vars.ISODate
	data["ISODateTime"] = vars.ISODateTime
	data["Unix"] = vars.Unix

	for k, v := range t.Variables {
		data[k] = v
	}

	for k, v := range customVars {
		data[k] = v
	}

	tmpl, err := template.New("template").Parse(t.Content)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

func (t *Template) Preview() (string, error) {
	return t.Render(nil)
}

func (s *TemplateStore) Export(name string) ([]byte, error) {
	t, err := s.Get(name)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(t, "", "  ")
}

func (s *TemplateStore) Import(data []byte) (*Template, error) {
	var t Template
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	if t.Name == "" {
		return nil, fmt.Errorf("template name is required")
	}

	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	if err := s.save(&t); err != nil {
		return nil, err
	}

	s.templates[t.Name] = &t
	return &t, nil
}

func (s *TemplateStore) ExportAll() ([]byte, error) {
	templates := s.List()
	return json.MarshalIndent(templates, "", "  ")
}

func (s *TemplateStore) ImportAll(data []byte) (int, error) {
	var templates []*Template
	if err := json.Unmarshal(data, &templates); err != nil {
		return 0, fmt.Errorf("parse templates: %w", err)
	}

	count := 0
	for _, t := range templates {
		if t.Name == "" {
			continue
		}

		t.CreatedAt = time.Now()
		t.UpdatedAt = time.Now()

		if err := s.save(t); err != nil {
			continue
		}

		s.templates[t.Name] = t
		count++
	}

	return count, nil
}

func GetDefaultVariables() map[string]string {
	return map[string]string{
		"Date":        "{{.Date}}",
		"Time":        "{{.Time}}",
		"DateTime":    "{{.DateTime}}",
		"Year":        "{{.Year}}",
		"Month":       "{{.Month}}",
		"Day":         "{{.Day}}",
		"Weekday":     "{{.Weekday}}",
		"Hour":        "{{.Hour}}",
		"Minute":      "{{.Minute}}",
		"Second":      "{{.Second}}",
		"Timestamp":   "{{.Timestamp}}",
		"ISODate":     "{{.ISODate}}",
		"ISODateTime": "{{.ISODateTime}}",
		"Unix":        "{{.Unix}}",
	}
}
