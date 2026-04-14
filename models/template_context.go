package models

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"net/mail"
	"net/url"
	"path"
	"text/template"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

// TemplateContext is an interface that allows both campaigns and email
// requests to have a PhishingTemplateContext generated for them.
type TemplateContext interface {
	getFromAddress() string
	getBaseURL() string
	getOrgId() int64
}

// PhishingTemplateContext is the context that is sent to any template, such
// as the email or landing page content.
type PhishingTemplateContext struct {
	From        string
	URL         string
	Tracker     string
	TrackingURL string
	RId         string
	BaseURL     string
	QRCode      string
	QRCodeURL   string
	OrgName     string
	OrgLogo     string
	OrgColor    string
	BaseRecipient
}

// NewPhishingTemplateContext returns a populated PhishingTemplateContext,
// parsing the correct fields from the provided TemplateContext and recipient.
func NewPhishingTemplateContext(ctx TemplateContext, r BaseRecipient, rid string) (PhishingTemplateContext, error) {
	f, err := mail.ParseAddress(ctx.getFromAddress())
	if err != nil {
		return PhishingTemplateContext{}, err
	}
	fn := f.Name
	if fn == "" {
		fn = f.Address
	}
	templateURL, err := ExecuteTemplate(ctx.getBaseURL(), r)
	if err != nil {
		return PhishingTemplateContext{}, err
	}

	// For the base URL, we'll reset the the path and the query
	// This will create a URL in the form of http://example.com
	baseURL, err := url.Parse(templateURL)
	if err != nil {
		return PhishingTemplateContext{}, err
	}
	baseURL.Path = ""
	baseURL.RawQuery = ""

	phishURL, _ := url.Parse(templateURL)
	q := phishURL.Query()
	q.Set(RecipientParameter, rid)
	phishURL.RawQuery = q.Encode()

	trackingURL, _ := url.Parse(templateURL)
	trackingURL.Path = path.Join(trackingURL.Path, "/track")
	trackingURL.RawQuery = q.Encode()

	qrCodeURL, _ := url.Parse(templateURL)
	qrCodeURL.Path = path.Join(qrCodeURL.Path, "/qr")
	qrCodeURL.RawQuery = q.Encode()

	// Generate inline QR code as a base64-encoded data URI
	qrCodeTag := generateQRCodeTag(phishURL.String())

	// Look up organization branding if available
	var orgName, orgLogo, orgColor string
	if orgId := ctx.getOrgId(); orgId > 0 {
		org, err := GetOrganization(orgId)
		if err == nil {
			orgName = org.Name
			orgLogo = org.LogoURL
			orgColor = org.PrimaryColor
		}
	}

	return PhishingTemplateContext{
		BaseRecipient: r,
		BaseURL:       baseURL.String(),
		URL:           phishURL.String(),
		TrackingURL:   trackingURL.String(),
		Tracker:       "<img alt='' style='display: none' src='" + trackingURL.String() + "'/>",
		From:          fn,
		RId:           rid,
		QRCode:        qrCodeTag,
		QRCodeURL:     qrCodeURL.String(),
		OrgName:       orgName,
		OrgLogo:       orgLogo,
		OrgColor:      orgColor,
	}, nil
}

// generateQRCodeTag creates a base64-encoded PNG QR code as an HTML img tag.
// The QR code encodes the provided URL so recipients can scan it.
func generateQRCodeTag(targetURL string) string {
	pngData, err := GenerateQRCodePNG(targetURL)
	if err != nil {
		return ""
	}
	b64 := base64.StdEncoding.EncodeToString(pngData)
	return fmt.Sprintf(`<img alt="QR Code" src="data:image/png;base64,%s" width="256" height="256"/>`, b64)
}

// GenerateQRCodePNG returns the raw PNG bytes of a 256x256 QR code encoding
// the given URL. It is used both for inline base64 tags and the /qr endpoint.
func GenerateQRCodePNG(targetURL string) ([]byte, error) {
	qrCode, err := qr.Encode(targetURL, qr.M, qr.Auto)
	if err != nil {
		return nil, err
	}
	qrCode, err = barcode.Scale(qrCode, 256, 256)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, qrCode); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExecuteTemplate creates a templated string based on the provided
// template body and data.
func ExecuteTemplate(text string, data interface{}) (string, error) {
	buff := bytes.Buffer{}
	tmpl, err := template.New("template").Parse(text)
	if err != nil {
		return buff.String(), err
	}
	err = tmpl.Execute(&buff, data)
	return buff.String(), err
}

// ValidationContext is used for validating templates and pages
type ValidationContext struct {
	FromAddress string
	BaseURL     string
}

func (vc ValidationContext) getFromAddress() string {
	return vc.FromAddress
}

func (vc ValidationContext) getBaseURL() string {
	return vc.BaseURL
}

func (vc ValidationContext) getOrgId() int64 {
	return 0
}

// ValidateTemplate ensures that the provided text in the page or template
// uses the supported template variables correctly.
func ValidateTemplate(text string) error {
	vc := ValidationContext{
		FromAddress: "foo@bar.com",
		BaseURL:     "http://example.com",
	}
	td := Result{
		BaseRecipient: BaseRecipient{
			Email:     "foo@bar.com",
			FirstName: "Foo",
			LastName:  "Bar",
			Position:  "Test",
		},
		RId: "123456",
	}
	ptx, err := NewPhishingTemplateContext(vc, td.BaseRecipient, td.RId)
	if err != nil {
		return err
	}
	_, err = ExecuteTemplate(text, ptx)
	if err != nil {
		return err
	}
	return nil
}
