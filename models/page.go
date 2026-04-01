package models

import (
	"errors"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	log "github.com/gophish/gophish/logger"
)

// Page contains the fields used for a Page model
type Page struct {
	Id                 int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId             int64     `json:"-" gorm:"column:user_id"`
	OrgId              int64     `json:"-" gorm:"column:org_id"`
	Name               string    `json:"name"`
	HTML               string    `json:"html" gorm:"column:html"`
	CaptureCredentials bool      `json:"capture_credentials" gorm:"column:capture_credentials"`
	CapturePasswords   bool      `json:"capture_passwords" gorm:"column:capture_passwords"`
	RedirectURL        string    `json:"redirect_url" gorm:"column:redirect_url"`
	ModifiedDate       time.Time `json:"modified_date"`
}

// ErrPageNameNotSpecified is thrown if the name of the landing page is blank.
var ErrPageNameNotSpecified = errors.New("Page Name not specified")

// parseHTML parses the page HTML on save to handle the
// capturing (or lack thereof!) of credentials and passwords
func (p *Page) parseHTML() error {
	d, err := goquery.NewDocumentFromReader(strings.NewReader(p.HTML))
	if err != nil {
		return err
	}
	forms := d.Find("form")
	forms.Each(func(i int, f *goquery.Selection) {
		// We always want the submitted events to be
		// sent to our server
		f.SetAttr("action", "")
		if p.CaptureCredentials {
			// If we don't want to capture passwords,
			// find all the password fields and remove the "name" attribute.
			if !p.CapturePasswords {
				inputs := f.Find("input")
				inputs.Each(func(j int, input *goquery.Selection) {
					if t, _ := input.Attr("type"); strings.EqualFold(t, "password") {
						input.RemoveAttr("name")
					}
				})
			} else {
				// If the user chooses to re-enable the capture passwords setting,
				// we need to re-add the name attribute
				inputs := f.Find("input")
				inputs.Each(func(j int, input *goquery.Selection) {
					if t, _ := input.Attr("type"); strings.EqualFold(t, "password") {
						input.SetAttr("name", "password")
					}
				})
			}
		} else {
			// Otherwise, remove the name from all
			// inputs.
			inputFields := f.Find("input")
			inputFields.Each(func(j int, input *goquery.Selection) {
				input.RemoveAttr("name")
			})
		}
	})
	p.HTML, err = d.Html()
	return err
}

// Validate ensures that a page contains the appropriate details
func (p *Page) Validate() error {
	if p.Name == "" {
		return ErrPageNameNotSpecified
	}
	// If the user specifies to capture passwords,
	// we automatically capture credentials
	if p.CapturePasswords && !p.CaptureCredentials {
		p.CaptureCredentials = true
	}
	if err := ValidateTemplate(p.HTML); err != nil {
		return err
	}
	if err := ValidateTemplate(p.RedirectURL); err != nil {
		return err
	}
	return p.parseHTML()
}

// GetPages returns the pages belonging to the given org scope.
func GetPages(scope OrgScope) ([]Page, error) {
	ps := []Page{}
	err := scopeQuery(db.Table("pages"), scope).Find(&ps).Error
	if err != nil {
		log.Error(err)
		return ps, err
	}
	return ps, err
}

// GetPage returns the page, if it exists, specified by the given id and org scope.
func GetPage(id int64, scope OrgScope) (Page, error) {
	p := Page{}
	err := scopeQuery(db.Where("id=?", id), scope).Find(&p).Error
	if err != nil {
		log.Error(err)
	}
	return p, err
}

// GetPageByName returns the page, if it exists, specified by the given name and org scope.
func GetPageByName(n string, scope OrgScope) (Page, error) {
	p := Page{}
	err := scopeQuery(db.Where("name=?", n), scope).Find(&p).Error
	if err != nil {
		log.Error(err)
	}
	return p, err
}

// PostPage creates a new page in the database.
func PostPage(p *Page) error {
	err := p.Validate()
	if err != nil {
		log.Error(err)
		return err
	}
	// Insert into the DB
	err = db.Save(p).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// PutPage edits an existing Page in the database.
// Per the PUT Method RFC, it presumes all data for a page is provided.
func PutPage(p *Page) error {
	err := p.Validate()
	if err != nil {
		return err
	}
	err = db.Where("id=?", p.Id).Save(p).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// DeletePage deletes an existing page in the database.
func DeletePage(id int64, scope OrgScope) error {
	err := scopeQuery(db.Where("id=?", id), scope).Delete(Page{}).Error
	if err != nil {
		log.Error(err)
	}
	return err
}
