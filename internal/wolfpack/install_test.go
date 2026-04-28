package wolfpack

import (
	"reflect"
	"testing"

	"github.com/ayushag-nv/wolfpack/internal/wolfpack/config"
)

func TestDeveloperDependencyInstallersIncludeCoreTools(t *testing.T) {
	var got []string
	for _, installer := range developerDependencyInstallers {
		got = append(got, installer.target)
	}
	want := []string{"uv", "ruff", "gh", "glab"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("developer dependency installers = %v, want %v", got, want)
	}
}

func TestInstallDeveloperDependenciesRunsInstallersInOrder(t *testing.T) {
	original := developerDependencyInstallers
	defer func() {
		developerDependencyInstallers = original
	}()

	var got []string
	developerDependencyInstallers = []dependencyInstaller{
		{target: "uv", label: "uv", run: func(config.Config) error {
			got = append(got, "uv")
			return nil
		}},
		{target: "gh", label: "GitHub CLI", run: func(config.Config) error {
			got = append(got, "gh")
			return nil
		}},
		{target: "glab", label: "GitLab CLI", run: func(config.Config) error {
			got = append(got, "glab")
			return nil
		}},
	}

	if err := installDeveloperDependencies(config.Config{}); err != nil {
		t.Fatal(err)
	}
	want := []string{"uv", "gh", "glab"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("install order = %v, want %v", got, want)
	}
}
