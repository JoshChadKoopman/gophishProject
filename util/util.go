package util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/csv"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/jordan-wright/email"
)

var (
	firstNameRegex = regexp.MustCompile(`(?i)first[\s_-]*name`)
	lastNameRegex  = regexp.MustCompile(`(?i)last[\s_-]*name`)
	emailRegex     = regexp.MustCompile(`(?i)email`)
	positionRegex  = regexp.MustCompile(`(?i)position`)
	phoneRegex     = regexp.MustCompile(`(?i)phone`)
)

// ParseMail takes in an HTTP Request and returns an Email object
func ParseMail(r *http.Request) (email.Email, error) {
	e := email.Email{}
	m, err := mail.ReadMessage(r.Body)
	if err != nil {
		return e, fmt.Errorf("error reading mail message: %w", err)
	}
	body, err := io.ReadAll(m.Body)
	if err != nil {
		return e, fmt.Errorf("error reading mail body: %w", err)
	}
	e.HTML = body
	return e, nil
}

// csvColumnMap holds the column indices detected from a CSV header row.
type csvColumnMap struct {
	firstName int
	lastName  int
	email     int
	position  int
	phone     int
}

// newCSVColumnMap initializes a column map with all indices unset (-1).
func newCSVColumnMap() csvColumnMap {
	return csvColumnMap{firstName: -1, lastName: -1, email: -1, position: -1, phone: -1}
}

// detectColumns scans the header record and returns a column map.
func detectColumns(header []string) csvColumnMap {
	cm := newCSVColumnMap()
	for i, v := range header {
		switch {
		case firstNameRegex.MatchString(v):
			cm.firstName = i
		case lastNameRegex.MatchString(v):
			cm.lastName = i
		case emailRegex.MatchString(v):
			cm.email = i
		case positionRegex.MatchString(v):
			cm.position = i
		case phoneRegex.MatchString(v):
			cm.phone = i
		}
	}
	return cm
}

// hasAnyColumn returns true if at least one column was detected.
func (cm csvColumnMap) hasAnyColumn() bool {
	return cm.firstName != -1 || cm.lastName != -1 || cm.email != -1 || cm.position != -1 || cm.phone != -1
}

// safeField returns the value at idx from record if in range, otherwise empty.
func safeField(record []string, idx int) string {
	if idx != -1 && len(record) > idx {
		return record[idx]
	}
	return ""
}

// extractTarget builds a Target from a CSV record using the column map.
// Returns the target and true if valid, or zero-value and false if the email is invalid.
func extractTarget(record []string, cm csvColumnMap) (models.Target, bool) {
	ea := ""
	if cm.email != -1 && len(record) > cm.email {
		csvEmail, err := mail.ParseAddress(record[cm.email])
		if err != nil {
			return models.Target{}, false
		}
		ea = csvEmail.Address
	}
	t := models.Target{
		BaseRecipient: models.BaseRecipient{
			FirstName: safeField(record, cm.firstName),
			LastName:  safeField(record, cm.lastName),
			Email:     ea,
			Position:  safeField(record, cm.position),
			Phone:     safeField(record, cm.phone),
		},
	}
	return t, true
}

// parseCSVPart reads targets from a single CSV multipart part.
func parseCSVPart(part io.Reader) []models.Target {
	var targets []models.Target
	reader := csv.NewReader(part)
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return targets
	}

	cm := detectColumns(header)
	if !cm.hasAnyColumn() {
		return targets
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if t, ok := extractTarget(record, cm); ok {
			targets = append(targets, t)
		}
	}
	return targets
}

// ParseCSV contains the logic to parse the user provided csv file containing Target entries
func ParseCSV(r *http.Request) ([]models.Target, error) {
	mr, err := r.MultipartReader()
	ts := []models.Target{}
	if err != nil {
		return ts, err
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		// Skip the "submit" part
		if part.FileName() == "" {
			continue
		}
		defer part.Close()
		ts = append(ts, parseCSVPart(part)...)
	}
	return ts, nil
}

// CheckAndCreateSSL is a helper to setup self-signed certificates for the administrative interface.
func CheckAndCreateSSL(cp string, kp string) error {
	// Check whether there is an existing SSL certificate and/or key, and if so, abort execution of this function
	if _, err := os.Stat(cp); !os.IsNotExist(err) {
		return nil
	}
	if _, err := os.Stat(kp); !os.IsNotExist(err) {
		return nil
	}

	log.Infof("Creating new self-signed certificates for administration interface")

	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return fmt.Errorf("error generating tls private key: %v", err)
	}

	notBefore := time.Now()
	// Generate a certificate that lasts for 10 years
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	if err != nil {
		return fmt.Errorf("tls certificate generation: failed to generate a random serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Gophish"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		return fmt.Errorf("tls certificate generation: failed to create certificate: %s", err)
	}

	certOut, err := os.Create(cp)
	if err != nil {
		return fmt.Errorf("tls certificate generation: failed to open %s for writing: %s", cp, err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.OpenFile(kp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("tls certificate generation: failed to open %s for writing", kp)
	}

	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("tls certificate generation: unable to marshal ECDSA private key: %v", err)
	}

	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
	keyOut.Close()

	log.Info("TLS Certificate Generation complete")
	return nil
}
