package format_test

import (
	"errors"
	"testing"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
	"github.com/addozhang/nexus-cli/internal/format"
)

func Test_Get_UnknownFormat(t *testing.T) {
	_, err := format.Get("rpm2")
	var nxErr *nxerrors.Error
	if !errors.As(err, &nxErr) || nxErr.Class() != nxerrors.ClassFlag {
		t.Fatalf("want ClassFlag error, got %v", err)
	}
}

func Test_Maven_Grammar(t *testing.T) {
	a, _ := format.Get("maven")

	t.Run("search free text", func(t *testing.T) {
		p, err := a.ParseSearch("commons")
		if err != nil || p.Q != "commons" {
			t.Errorf("got %+v, err %v", p, err)
		}
	})
	t.Run("search group:artifact", func(t *testing.T) {
		p, err := a.ParseSearch("com.example:foo")
		if err != nil || p.Group != "com.example" || p.Name != "foo" || p.Version != "" {
			t.Errorf("got %+v, err %v", p, err)
		}
	})
	t.Run("search with version", func(t *testing.T) {
		p, err := a.ParseSearch("com.example:foo:1.2.0")
		if err != nil || p.Version != "1.2.0" {
			t.Errorf("got %+v, err %v", p, err)
		}
	})
	t.Run("empty segment rejected", func(t *testing.T) {
		_, err := a.ParseSearch("com.example::1.0")
		var nxErr *nxerrors.Error
		if !errors.As(err, &nxErr) || nxErr.Class() != nxerrors.ClassCoordinate {
			t.Errorf("want ClassCoordinate, got %v", err)
		}
	})
	t.Run("component requires group separator", func(t *testing.T) {
		_, err := a.ParseComponent("foo", false)
		if err == nil {
			t.Error("expected error for missing group")
		}
	})
	t.Run("info requires version", func(t *testing.T) {
		_, err := a.ParseComponent("com.example:foo", true)
		if err == nil {
			t.Error("expected error for missing version")
		}
	})
	t.Run("full coordinate ok", func(t *testing.T) {
		ref, err := a.ParseComponent("com.example:foo:1.2.0", true)
		if err != nil || ref.Group != "com.example" || ref.Name != "foo" || ref.Version != "1.2.0" {
			t.Errorf("got %+v, err %v", ref, err)
		}
	})
}

func Test_SimpleFormats_Grammar(t *testing.T) {
	for _, name := range []string{"npm", "pypi", "cargo", "go"} {
		t.Run(name, func(t *testing.T) {
			a, _ := format.Get(name)

			p, err := a.ParseSearch("lodash@4.17.21")
			if err != nil || p.Name != "lodash" || p.Version != "4.17.21" || p.Q != "" {
				t.Errorf("ParseSearch versioned: got %+v, err %v", p, err)
			}
			p, err = a.ParseSearch("requests")
			if err != nil || p.Q != "requests" {
				t.Errorf("ParseSearch plain: got %+v, err %v", p, err)
			}

			_, err = a.ParseComponent("left-pad", true)
			var nxErr *nxerrors.Error
			if !errors.As(err, &nxErr) || nxErr.Class() != nxerrors.ClassCoordinate {
				t.Errorf("info without version should fail, got %v", err)
			}

			ref, err := a.ParseComponent("left-pad@1.3.0", true)
			if err != nil || ref.Name != "left-pad" || ref.Version != "1.3.0" {
				t.Errorf("ParseComponent: got %+v, err %v", ref, err)
			}

			ref, err = a.ParseComponent("left-pad", false)
			if err != nil || ref.Name != "left-pad" {
				t.Errorf("versions without version ok: got %+v, err %v", ref, err)
			}
		})
	}
}

func Test_Docker_Grammar(t *testing.T) {
	a, _ := format.Get("docker")

	t.Run("image with tag", func(t *testing.T) {
		ref, err := a.ParseComponent("myteam/app:1.0", false)
		if err != nil || ref.Name != "myteam/app" || ref.Version != "1.0" {
			t.Errorf("got %+v, err %v", ref, err)
		}
	})
	t.Run("registry host port preserved", func(t *testing.T) {
		ref, err := a.ParseComponent("localhost:5000/app:v2", false)
		if err != nil || ref.Name != "localhost:5000/app" || ref.Version != "v2" {
			t.Errorf("got %+v, err %v", ref, err)
		}
	})
	t.Run("bare image is free text search", func(t *testing.T) {
		p, err := a.ParseSearch("nginx")
		if err != nil || p.Q != "nginx" {
			t.Errorf("got %+v, err %v", p, err)
		}
	})
}
