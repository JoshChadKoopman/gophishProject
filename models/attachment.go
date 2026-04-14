package models

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"io"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// Attachment contains the fields and methods for
// an email attachment
type Attachment struct {
	Id          int64  `json:"-"`
	TemplateId  int64  `json:"-"`
	Content     string `json:"content"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	vanillaFile bool   // Vanilla file has no template variables
}

// Validate ensures that the provided attachment uses the supported template variables correctly.
func (a Attachment) Validate() error {
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
	_, err = a.ApplyTemplate(ptx)
	return err
}

// ApplyTemplate parses different attachment files and applies the supplied phishing template.
func (a *Attachment) ApplyTemplate(ptx PhishingTemplateContext) (io.Reader, error) {

	decodedAttachment := base64.NewDecoder(base64.StdEncoding, strings.NewReader(a.Content))

	// If we've already determined there are no template variables in this attachment return it immediately
	if a.vanillaFile {
		return decodedAttachment, nil
	}

	fileExtension := filepath.Ext(a.Name)

	switch fileExtension {
	case ".docx", ".docm", ".pptx", ".xlsx", ".xlsm":
		return a.applyTemplateToOfficeDoc(decodedAttachment, ptx)
	case ".txt", ".html", ".ics":
		return a.applyTemplateToTextFile(decodedAttachment, ptx)
	default:
		return decodedAttachment, nil
	}
}

// urlEscapedTemplateVarRegex matches Word-URL-escaped template variables like %7b%7b.Foo%7d%7d.
var urlEscapedTemplateVarRegex = regexp.MustCompile("%7b%7b.([a-zA-Z]+)%7d%7d")

// applyTemplateToOfficeDoc handles zip-based Office formats (.docx, .pptx, .xlsx, etc.).
func (a *Attachment) applyTemplateToOfficeDoc(decoded io.Reader, ptx PhishingTemplateContext) (io.Reader, error) {
	b := new(bytes.Buffer)
	b.ReadFrom(decoded)
	zipReader, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err != nil {
		return nil, err
	}

	newZipArchive := new(bytes.Buffer)
	zipWriter := zip.NewWriter(newZipArchive)

	a.vanillaFile = true
	for _, zipFile := range zipReader.File {
		tFile, processErr := a.processZipEntry(zipFile, ptx)
		if processErr != nil {
			zipWriter.Close()
			return nil, processErr
		}
		newZipFile, createErr := zipWriter.Create(zipFile.Name)
		if createErr != nil {
			zipWriter.Close()
			return nil, createErr
		}
		if _, writeErr := newZipFile.Write([]byte(tFile)); writeErr != nil {
			zipWriter.Close()
			return nil, writeErr
		}
	}
	zipWriter.Close()
	return bytes.NewReader(newZipArchive.Bytes()), nil
}

// processZipEntry reads a single file from a zip archive and applies templates
// to XML/RELS entries.
func (a *Attachment) processZipEntry(zipFile *zip.File, ptx PhishingTemplateContext) (string, error) {
	ff, err := zipFile.Open()
	if err != nil {
		return "", err
	}
	defer ff.Close()
	contents, err := io.ReadAll(ff)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(zipFile.Name)
	if ext != ".xml" && ext != ".rels" {
		return string(contents), nil
	}
	// Unescape Word-URL-encoded template variables.
	contents = urlEscapedTemplateVarRegex.ReplaceAllFunc(contents, func(m []byte) []byte {
		d, unescErr := url.QueryUnescape(string(m))
		if unescErr != nil {
			return m
		}
		return []byte(d)
	})
	tFile, err := ExecuteTemplate(string(contents), ptx)
	if err != nil {
		return "", err
	}
	if tFile != string(contents) {
		a.vanillaFile = false
	}
	return tFile, nil
}

// applyTemplateToTextFile handles plain-text-like attachments (.txt, .html, .ics).
func (a *Attachment) applyTemplateToTextFile(decoded io.Reader, ptx PhishingTemplateContext) (io.Reader, error) {
	b, err := io.ReadAll(decoded)
	if err != nil {
		return nil, err
	}
	processedAttachment, err := ExecuteTemplate(string(b), ptx)
	if err != nil {
		return nil, err
	}
	if processedAttachment == string(b) {
		a.vanillaFile = true
	}
	return strings.NewReader(processedAttachment), nil
}
