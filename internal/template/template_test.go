package template

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewTemplateStore(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	if store == nil {
		t.Fatal("store is nil")
	}

	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0", store.Count())
	}
}

func TestTemplateSave(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	tmpl, err := store.Save("test-template", "Hello {{.Name}}!", "greeting", "A greeting template", map[string]string{"Name": "World"})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if tmpl.Name != "test-template" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "test-template")
	}

	if tmpl.Content != "Hello {{.Name}}!" {
		t.Errorf("Content = %q, want %q", tmpl.Content, "Hello {{.Name}}!")
	}

	if tmpl.Category != "greeting" {
		t.Errorf("Category = %q, want %q", tmpl.Category, "greeting")
	}

	if store.Count() != 1 {
		t.Errorf("Count() = %d, want 1", store.Count())
	}

	filePath := filepath.Join(dir, "test-template.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("template file was not created")
	}
}

func TestTemplateGet(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	_, err = store.Save("test", "content", "", "", nil)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	tmpl, err := store.Get("test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if tmpl.Name != "test" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "test")
	}

	_, err = store.Get("nonexistent")
	if err == nil {
		t.Error("Get() should return error for nonexistent template")
	}
}

func TestTemplateDelete(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	_, err = store.Save("test", "content", "", "", nil)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := store.Delete("test"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if store.Count() != 0 {
		t.Errorf("Count() = %d, want 0", store.Count())
	}

	_, err = store.Get("test")
	if err == nil {
		t.Error("Get() should return error after delete")
	}

	err = store.Delete("nonexistent")
	if err == nil {
		t.Error("Delete() should return error for nonexistent template")
	}
}

func TestTemplateList(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	_, _ = store.Save("alpha", "content", "cat1", "", nil)
	_, _ = store.Save("beta", "content", "cat2", "", nil)
	_, _ = store.Save("gamma", "content", "cat1", "", nil)

	templates := store.List()
	if len(templates) != 3 {
		t.Errorf("List() returned %d templates, want 3", len(templates))
	}

	if templates[0].Category != "cat1" {
		t.Errorf("first template category = %q, want %q", templates[0].Category, "cat1")
	}
}

func TestTemplateListByCategory(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	_, _ = store.Save("alpha", "content", "cat1", "", nil)
	_, _ = store.Save("beta", "content", "cat2", "", nil)
	_, _ = store.Save("gamma", "content", "cat1", "", nil)

	templates := store.ListByCategory("cat1")
	if len(templates) != 2 {
		t.Errorf("ListByCategory(cat1) returned %d templates, want 2", len(templates))
	}

	templates = store.ListByCategory("cat2")
	if len(templates) != 1 {
		t.Errorf("ListByCategory(cat2) returned %d templates, want 1", len(templates))
	}
}

func TestTemplateCategories(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	_, _ = store.Save("t1", "content", "cat1", "", nil)
	_, _ = store.Save("t2", "content", "cat2", "", nil)
	_, _ = store.Save("t3", "content", "cat1", "", nil)

	categories := store.Categories()
	if len(categories) != 2 {
		t.Errorf("Categories() returned %d categories, want 2", len(categories))
	}
}

func TestTemplateUpdate(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	_, err = store.Save("test", "original", "cat1", "desc", nil)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	tmpl, err := store.Update("test", "updated", "cat2", "new desc", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if tmpl.Content != "updated" {
		t.Errorf("Content = %q, want %q", tmpl.Content, "updated")
	}

	if tmpl.Category != "cat2" {
		t.Errorf("Category = %q, want %q", tmpl.Category, "cat2")
	}

	if tmpl.UpdatedAt.Before(tmpl.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}
}

func TestVariableSubstitution(t *testing.T) {
	tmpl := &Template{
		Name:    "test",
		Content: "Date: {{.Date}}, Time: {{.Time}}, Custom: {{.CustomVar}}",
	}

	result, err := tmpl.Render(map[string]string{"CustomVar": "custom-value"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	now := time.Now()
	expectedDate := now.Format("2006-01-02")
	expectedTime := now.Format("15:04:05")

	if !containsAll(result, expectedDate, expectedTime, "custom-value") {
		t.Errorf("Render() = %q, should contain %q, %q, %q", result, expectedDate, expectedTime, "custom-value")
	}
}

func TestVariableSubstitutionWithDefaults(t *testing.T) {
	tmpl := &Template{
		Name:      "test",
		Content:   "Hello {{.Name}}!",
		Variables: map[string]string{"Name": "World"},
	}

	result, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if result != "Hello World!" {
		t.Errorf("Render() = %q, want %q", result, "Hello World!")
	}
}

func TestVariableSubstitutionOverride(t *testing.T) {
	tmpl := &Template{
		Name:      "test",
		Content:   "Hello {{.Name}}!",
		Variables: map[string]string{"Name": "World"},
	}

	result, err := tmpl.Render(map[string]string{"Name": "Custom"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if result != "Hello Custom!" {
		t.Errorf("Render() = %q, want %q", result, "Hello Custom!")
	}
}

func TestAllBuiltInVariables(t *testing.T) {
	tmpl := &Template{
		Name: "test",
		Content: `Date: {{.Date}}
Time: {{.Time}}
DateTime: {{.DateTime}}
Year: {{.Year}}
Month: {{.Month}}
Day: {{.Day}}
Weekday: {{.Weekday}}
Hour: {{.Hour}}
Minute: {{.Minute}}
Second: {{.Second}}
Timestamp: {{.Timestamp}}
ISODate: {{.ISODate}}
ISODateTime: {{.ISODateTime}}
Unix: {{.Unix}}`,
	}

	result, err := tmpl.Render(nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	now := time.Now()
	expectedValues := []string{
		now.Format("2006-01-02"),
		now.Format("15:04:05"),
		now.Format("2006"),
		now.Format("01"),
		now.Format("02"),
		now.Weekday().String(),
		now.Format("15"),
		now.Format("04"),
		now.Format("05"),
		now.Format(time.RFC3339),
	}

	for _, expected := range expectedValues {
		if !contains(result, expected) {
			t.Errorf("Render() result should contain %q", expected)
		}
	}
}

func TestTemplatePreview(t *testing.T) {
	tmpl := &Template{
		Name:      "test",
		Content:   "Hello {{.Name}}!",
		Variables: map[string]string{"Name": "Preview"},
	}

	result, err := tmpl.Preview()
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	if result != "Hello Preview!" {
		t.Errorf("Preview() = %q, want %q", result, "Hello Preview!")
	}
}

func TestTemplateExport(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	_, _ = store.Save("test", "content", "cat", "desc", map[string]string{"key": "value"})

	data, err := store.Export("test")
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	var exported Template
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("unmarshal exported data: %v", err)
	}

	if exported.Name != "test" {
		t.Errorf("exported Name = %q, want %q", exported.Name, "test")
	}
}

func TestTemplateImport(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	templateData := Template{
		Name:        "imported",
		Content:     "imported content",
		Category:    "imported-cat",
		Description: "imported desc",
		Variables:   map[string]string{"key": "value"},
	}

	data, err := json.Marshal(templateData)
	if err != nil {
		t.Fatalf("marshal template: %v", err)
	}

	tmpl, err := store.Import(data)
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if tmpl.Name != "imported" {
		t.Errorf("imported Name = %q, want %q", tmpl.Name, "imported")
	}

	if store.Count() != 1 {
		t.Errorf("Count() = %d, want 1", store.Count())
	}
}

func TestTemplateExportAll(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	_, _ = store.Save("t1", "content1", "cat1", "", nil)
	_, _ = store.Save("t2", "content2", "cat2", "", nil)

	data, err := store.ExportAll()
	if err != nil {
		t.Fatalf("ExportAll() error = %v", err)
	}

	var templates []*Template
	if err := json.Unmarshal(data, &templates); err != nil {
		t.Fatalf("unmarshal exported data: %v", err)
	}

	if len(templates) != 2 {
		t.Errorf("exported %d templates, want 2", len(templates))
	}
}

func TestTemplateImportAll(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	templates := []*Template{
		{Name: "imported1", Content: "content1", Category: "cat1"},
		{Name: "imported2", Content: "content2", Category: "cat2"},
	}

	data, err := json.Marshal(templates)
	if err != nil {
		t.Fatalf("marshal templates: %v", err)
	}

	count, err := store.ImportAll(data)
	if err != nil {
		t.Fatalf("ImportAll() error = %v", err)
	}

	if count != 2 {
		t.Errorf("ImportAll() = %d, want 2", count)
	}

	if store.Count() != 2 {
		t.Errorf("Count() = %d, want 2", store.Count())
	}
}

func TestTemplateLoadFromFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "x-cli-templates-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	templateData := Template{
		Name:        "file-template",
		Content:     "content from file",
		Category:    "file-cat",
		Description: "template loaded from file",
		Variables:   map[string]string{"var1": "value1"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	data, err := json.MarshalIndent(templateData, "", "  ")
	if err != nil {
		t.Fatalf("marshal template: %v", err)
	}

	filePath := filepath.Join(dir, "file-template.json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("write template file: %v", err)
	}

	store, err := NewTemplateStoreWithDir(dir)
	if err != nil {
		t.Fatalf("NewTemplateStoreWithDir() error = %v", err)
	}

	tmpl, err := store.Get("file-template")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if tmpl.Name != "file-template" {
		t.Errorf("Name = %q, want %q", tmpl.Name, "file-template")
	}

	if tmpl.Content != "content from file" {
		t.Errorf("Content = %q, want %q", tmpl.Content, "content from file")
	}
}

func TestInvalidTemplateSyntax(t *testing.T) {
	tmpl := &Template{
		Name:    "invalid",
		Content: "Hello {{.Name",
	}

	_, err := tmpl.Render(nil)
	if err == nil {
		t.Error("Render() should return error for invalid template syntax")
	}
}

func TestGetDefaultVariables(t *testing.T) {
	vars := GetDefaultVariables()

	expectedKeys := []string{
		"Date", "Time", "DateTime", "Year", "Month", "Day",
		"Weekday", "Hour", "Minute", "Second",
		"Timestamp", "ISODate", "ISODateTime", "Unix",
	}

	for _, key := range expectedKeys {
		if _, ok := vars[key]; !ok {
			t.Errorf("GetDefaultVariables() missing key %q", key)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsAll(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !contains(s, substr) {
			return false
		}
	}
	return true
}
