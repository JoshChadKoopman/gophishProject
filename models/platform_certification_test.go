package models

import (
	"testing"
)

func TestPlatformCertificationConstants(t *testing.T) {
	if PlatformCertStatusCertified != "certified" {
		t.Fatalf("expected 'certified', got %q", PlatformCertStatusCertified)
	}
	if PlatformCertStatusAligned != "aligned" {
		t.Fatalf("expected 'aligned', got %q", PlatformCertStatusAligned)
	}
	if PlatformCertStatusInProgress != "in_progress" {
		t.Fatalf("expected 'in_progress', got %q", PlatformCertStatusInProgress)
	}
}

func TestBuiltInPlatformCertificationsNotEmpty(t *testing.T) {
	if len(BuiltInPlatformCertifications) == 0 {
		t.Fatal("expected at least 1 built-in platform certification")
	}
}

func TestBuiltInPlatformCertificationsHaveRequiredFields(t *testing.T) {
	for _, cert := range BuiltInPlatformCertifications {
		if cert.Slug == "" {
			t.Fatal("platform cert has empty slug")
		}
		if cert.Name == "" {
			t.Fatalf("platform cert %q has empty name", cert.Slug)
		}
		if cert.Standard == "" {
			t.Fatalf("platform cert %q has empty standard", cert.Slug)
		}
		if cert.Status == "" {
			t.Fatalf("platform cert %q has empty status", cert.Slug)
		}
		if cert.Description == "" {
			t.Fatalf("platform cert %q has empty description", cert.Slug)
		}
		if cert.DeploymentModel == "" {
			t.Fatalf("platform cert %q has empty deployment model", cert.Slug)
		}
		if len(cert.KeyControls) == 0 {
			t.Fatalf("platform cert %q has no key controls", cert.Slug)
		}
	}
}

func TestBuiltInPlatformCertificationsUniqueSlugs(t *testing.T) {
	slugs := map[string]bool{}
	for _, cert := range BuiltInPlatformCertifications {
		if slugs[cert.Slug] {
			t.Fatalf("duplicate platform cert slug: %q", cert.Slug)
		}
		slugs[cert.Slug] = true
	}
}

func TestPlatformCertificationsContainISO(t *testing.T) {
	found := false
	for _, cert := range BuiltInPlatformCertifications {
		if cert.Slug == "iso27001" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected ISO 27001 certification to be in built-in list")
	}
}

func TestPlatformCertificationsContainSOC2(t *testing.T) {
	found := false
	for _, cert := range BuiltInPlatformCertifications {
		if cert.Slug == "soc2-type2" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected SOC 2 Type II certification to be in built-in list")
	}
}
