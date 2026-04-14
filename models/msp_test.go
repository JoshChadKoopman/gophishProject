package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupMSPTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	return func() { /* in-memory DB cleaned up automatically */ }
}

func TestMSPFeatureConstants(t *testing.T) {
	if FeatureMSPPartnerPortal != "msp_partner_portal" {
		t.Fatalf("expected 'msp_partner_portal', got %q", FeatureMSPPartnerPortal)
	}
	if FeatureMSPMultiClient != "msp_multi_client" {
		t.Fatalf("expected 'msp_multi_client', got %q", FeatureMSPMultiClient)
	}
}

func TestCreateAndGetMSPPartner(t *testing.T) {
	teardown := setupMSPTest(t)
	defer teardown()

	p := &MSPPartner{
		Name:         "Test Partner",
		Slug:         "test-partner",
		ContactEmail: "admin@partner.com",
		MaxClients:   20,
		IsActive:     true,
	}
	err := PostMSPPartner(p)
	if err != nil {
		t.Fatalf("PostMSPPartner: %v", err)
	}
	if p.Id == 0 {
		t.Fatal("expected partner to have an ID")
	}

	got, err := GetMSPPartner(p.Id)
	if err != nil {
		t.Fatalf("GetMSPPartner: %v", err)
	}
	if got.Name != "Test Partner" {
		t.Fatalf("expected 'Test Partner', got %q", got.Name)
	}
	if got.Slug != "test-partner" {
		t.Fatalf("expected 'test-partner', got %q", got.Slug)
	}
}

func TestGetMSPPartnerNotFound(t *testing.T) {
	teardown := setupMSPTest(t)
	defer teardown()

	_, err := GetMSPPartner(99999)
	if err == nil {
		t.Fatal("expected error for non-existent partner")
	}
}

func TestGetMSPPartners(t *testing.T) {
	teardown := setupMSPTest(t)
	defer teardown()

	PostMSPPartner(&MSPPartner{Name: "P1", Slug: "p1", IsActive: true})
	PostMSPPartner(&MSPPartner{Name: "P2", Slug: "p2", IsActive: true})

	partners, err := GetMSPPartners()
	if err != nil {
		t.Fatalf("GetMSPPartners: %v", err)
	}
	if len(partners) < 2 {
		t.Fatalf("expected at least 2 partners, got %d", len(partners))
	}
}

func TestAddMSPPartnerClient(t *testing.T) {
	teardown := setupMSPTest(t)
	defer teardown()

	PostMSPPartner(&MSPPartner{Name: "P1", Slug: "p1-client", IsActive: true, MaxClients: 10})

	_, err := AddMSPPartnerClient(1, 1)
	if err != nil {
		t.Fatalf("AddMSPPartnerClient: %v", err)
	}

	clients, err := GetMSPPartnerClients(1)
	if err != nil {
		t.Fatalf("GetMSPPartnerClients: %v", err)
	}
	if len(clients) < 1 {
		t.Fatal("expected at least 1 client")
	}
}

func TestWhiteLabelConfigCRUD(t *testing.T) {
	teardown := setupMSPTest(t)
	defer teardown()

	wl := &WhiteLabelConfig{
		OrgId:        1,
		CompanyName:  "Acme Corp",
		PrimaryColor: "#ff0000",
		IsActive:     true,
	}
	if err := SaveWhiteLabelConfig(wl); err != nil {
		t.Fatalf("SaveWhiteLabelConfig: %v", err)
	}

	got, err := GetWhiteLabelConfig(1)
	if err != nil {
		t.Fatalf("GetWhiteLabelConfig: %v", err)
	}
	if got.CompanyName != "Acme Corp" {
		t.Fatalf("expected 'Acme Corp', got %q", got.CompanyName)
	}
	if got.PrimaryColor != "#ff0000" {
		t.Fatalf("expected '#ff0000', got %q", got.PrimaryColor)
	}
}
