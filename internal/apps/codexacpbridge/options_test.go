package codexacp

import "testing"

func TestCodexAppConfigWithModel(t *testing.T) {
	cfg := codexAppConfig{Model: "default-model"}

	withOverride := cfg.withModel(" custom-model ")
	if got, want := withOverride.Model, "custom-model"; got != want {
		t.Fatalf("withModel(override).Model = %q, want %q", got, want)
	}

	withEmpty := cfg.withModel("  ")
	if got, want := withEmpty.Model, "default-model"; got != want {
		t.Fatalf("withModel(empty).Model = %q, want %q", got, want)
	}
}

func TestCodexAppConfigCloneDeepCopiesMutableFields(t *testing.T) {
	ephemeral := false
	cfg := codexAppConfig{
		Config:    map[string]any{"k": "v"},
		Ephemeral: &ephemeral,
	}

	cloned := cfg.clone()
	if cloned.Config == nil {
		t.Fatal("clone().Config = nil, want non-nil")
	}
	if cloned.Ephemeral == nil {
		t.Fatal("clone().Ephemeral = nil, want non-nil")
	}
	if cloned.Ephemeral == cfg.Ephemeral {
		t.Fatal("clone().Ephemeral points to original pointer, want copy")
	}

	cloned.Config["k"] = "options-clone-updated"
	if got, want := cfg.Config["k"], "v"; got != want {
		t.Fatalf("original Config after clone mutation = %#v, want %q", got, want)
	}

	*cloned.Ephemeral = true
	if *cfg.Ephemeral {
		t.Fatal("original Ephemeral changed after clone mutation, want independent copy")
	}
}

func TestValidateEnumValue(t *testing.T) {
	allowed := map[string]struct{}{
		"one": {},
		"two": {},
	}

	if err := validateEnumValue("example", "", allowed); err != nil {
		t.Fatalf("validateEnumValue(empty) error = %v, want nil", err)
	}
	if err := validateEnumValue("example", " two ", allowed); err != nil {
		t.Fatalf("validateEnumValue(valid) error = %v, want nil", err)
	}
	if err := validateEnumValue("example", "three", allowed); err == nil {
		t.Fatal("validateEnumValue(invalid) error = nil, want non-nil")
	}
}

func TestCloneMap(t *testing.T) {
	if got := cloneMap(nil); got != nil {
		t.Fatalf("cloneMap(nil) = %#v, want nil", got)
	}

	src := map[string]any{"k": "v"}
	cloned := cloneMap(src)
	if cloned == nil {
		t.Fatal("cloneMap(src) = nil, want non-nil")
	}

	cloned["k"] = "options-clone-map-updated"
	if got, want := src["k"], "v"; got != want {
		t.Fatalf("source map after clone mutation = %#v, want %q", got, want)
	}
}
