package models

import (
	"errors"
	"fmt"
	"net/mail"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Group contains the fields needed for a user -> group mapping
// Groups contain 1..* Targets
type Group struct {
	Id           int64     `json:"id"`
	UserId       int64     `json:"-"`
	OrgId        int64     `json:"-" gorm:"column:org_id"`
	Name         string    `json:"name"`
	ModifiedDate time.Time `json:"modified_date"`
	Targets      []Target  `json:"targets" sql:"-"`
}

// GroupSummaries is a struct representing the overview of Groups.
type GroupSummaries struct {
	Total  int64          `json:"total"`
	Groups []GroupSummary `json:"groups"`
}

// GroupSummary represents a summary of the Group model. The only
// difference is that, instead of listing the Targets (which could be expensive
// for large groups), it lists the target count.
type GroupSummary struct {
	Id           int64     `json:"id"`
	Name         string    `json:"name"`
	ModifiedDate time.Time `json:"modified_date"`
	NumTargets   int64     `json:"num_targets"`
}

// GroupTarget is used for a many-to-many relationship between 1..* Groups and 1..* Targets
type GroupTarget struct {
	GroupId  int64 `json:"-"`
	TargetId int64 `json:"-"`
}

// Target contains the fields needed for individual targets specified by the user
// Groups contain 1..* Targets, but 1 Target may belong to 1..* Groups
type Target struct {
	Id int64 `json:"-"`
	BaseRecipient
}

// BaseRecipient contains the fields for a single recipient. This is the base
// struct used in members of groups and campaign results.
type BaseRecipient struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Position  string `json:"position"`
	Phone     string `json:"phone"`
}

// FormatAddress returns the email address to use in the "To" header of the email
func (r *BaseRecipient) FormatAddress() string {
	addr := r.Email
	if r.FirstName != "" && r.LastName != "" {
		a := &mail.Address{
			Name:    fmt.Sprintf("%s %s", r.FirstName, r.LastName),
			Address: r.Email,
		}
		addr = a.String()
	}
	return addr
}

// FormatAddress returns the email address to use in the "To" header of the email
func (t *Target) FormatAddress() string {
	addr := t.Email
	if t.FirstName != "" && t.LastName != "" {
		a := &mail.Address{
			Name:    fmt.Sprintf("%s %s", t.FirstName, t.LastName),
			Address: t.Email,
		}
		addr = a.String()
	}
	return addr
}

// ErrEmailNotSpecified is thrown when no email is specified for the Target
var ErrEmailNotSpecified = errors.New("No email address specified")

// ErrGroupNameNotSpecified is thrown when a group name is not specified
var ErrGroupNameNotSpecified = errors.New("Group name not specified")

// ErrNoTargetsSpecified is thrown when no targets are specified by the user
var ErrNoTargetsSpecified = errors.New("No targets specified")

// queryWhereGroupID is the shared WHERE clause for group_id lookups.
const queryWhereGroupID = "group_id=?"

// Validate performs validation on a group given by the user
func (g *Group) Validate() error {
	switch {
	case g.Name == "":
		return ErrGroupNameNotSpecified
	case len(g.Targets) == 0:
		return ErrNoTargetsSpecified
	}
	return nil
}

// GetGroups returns the groups belonging to the given org scope.
func GetGroups(scope OrgScope) ([]Group, error) {
	gs := []Group{}
	query := scopeQuery(db.Table("groups"), scope)
	err := query.Find(&gs).Error
	if err != nil {
		log.Error(err)
		return gs, err
	}
	for i := range gs {
		gs[i].Targets, err = GetTargets(gs[i].Id)
		if err != nil {
			log.Error(err)
		}
	}
	return gs, nil
}

// GetGroupSummaries returns the summaries for the groups
// belonging to the given org scope.
func GetGroupSummaries(scope OrgScope) (GroupSummaries, error) {
	gs := GroupSummaries{}
	query := scopeQuery(db.Table("groups"), scope)
	err := query.Select("id, name, modified_date").Scan(&gs.Groups).Error
	if err != nil {
		log.Error(err)
		return gs, err
	}
	for i := range gs.Groups {
		query = db.Table("group_targets").Where(queryWhereGroupID, gs.Groups[i].Id)
		err = query.Count(&gs.Groups[i].NumTargets).Error
		if err != nil {
			return gs, err
		}
	}
	gs.Total = int64(len(gs.Groups))
	return gs, nil
}

// GetGroup returns the group, if it exists, specified by the given id and org scope.
func GetGroup(id int64, scope OrgScope) (Group, error) {
	g := Group{}
	query := scopeQuery(db.Where("id=?", id), scope)
	err := query.Find(&g).Error
	if err != nil {
		log.Error(err)
		return g, err
	}
	g.Targets, err = GetTargets(g.Id)
	if err != nil {
		log.Error(err)
	}
	return g, nil
}

// GetGroupSummary returns the summary for the requested group
func GetGroupSummary(id int64, scope OrgScope) (GroupSummary, error) {
	g := GroupSummary{}
	query := scopeQuery(db.Table("groups").Where("id=?", id), scope)
	err := query.Select("id, name, modified_date").Scan(&g).Error
	if err != nil {
		log.Error(err)
		return g, err
	}
	query = db.Table("group_targets").Where(queryWhereGroupID, id)
	err = query.Count(&g.NumTargets).Error
	if err != nil {
		return g, err
	}
	return g, nil
}

// GetGroupByName returns the group, if it exists, specified by the given name and org scope.
func GetGroupByName(n string, scope OrgScope) (Group, error) {
	g := Group{}
	query := scopeQuery(db.Where("name=?", n), scope)
	err := query.Find(&g).Error
	if err != nil {
		log.Error(err)
		return g, err
	}
	g.Targets, err = GetTargets(g.Id)
	if err != nil {
		log.Error(err)
	}
	return g, err
}

// PostGroup creates a new group in the database.
func PostGroup(g *Group) error {
	if err := g.Validate(); err != nil {
		return err
	}
	// Insert the group into the DB
	tx := db.Begin()
	err := tx.Save(g).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	for _, t := range g.Targets {
		err = insertTargetIntoGroup(tx, t, g.Id)
		if err != nil {
			tx.Rollback()
			log.Error(err)
			return err
		}
	}
	err = tx.Commit().Error
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return err
	}
	return nil
}

// PutGroup updates the given group if found in the database.
func PutGroup(g *Group) error {
	if err := g.Validate(); err != nil {
		return err
	}
	ts, err := GetTargets(g.Id)
	if err != nil {
		log.WithFields(logrus.Fields{"group_id": g.Id}).Error("Error getting targets from group")
		return err
	}

	cacheNew := buildEmailCache(g.Targets)
	cacheExisting := buildEmailCache(ts)

	tx := db.Begin()
	removeStaleTargets(tx, ts, cacheNew, g.Id)

	if err := upsertNewTargets(tx, g.Targets, cacheExisting, g.Id); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Save(g).Error; err != nil {
		log.Error(err)
		return err
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

func buildEmailCache(targets []Target) map[string]int64 {
	cache := make(map[string]int64, len(targets))
	for _, t := range targets {
		cache[t.Email] = t.Id
	}
	return cache
}

func removeStaleTargets(tx *gorm.DB, existing []Target, cacheNew map[string]int64, groupId int64) {
	for _, t := range existing {
		if _, ok := cacheNew[t.Email]; ok {
			continue
		}
		if err := tx.Where("group_id=? and target_id=?", groupId, t.Id).Delete(&GroupTarget{}).Error; err != nil {
			log.WithFields(logrus.Fields{"email": t.Email}).Error("Error deleting email")
		}
	}
}

func upsertNewTargets(tx *gorm.DB, targets []Target, cacheExisting map[string]int64, groupId int64) error {
	for _, nt := range targets {
		if id, ok := cacheExisting[nt.Email]; ok {
			nt.Id = id
			if err := UpdateTarget(tx, nt); err != nil {
				log.Error(err)
				return err
			}
			continue
		}
		if err := insertTargetIntoGroup(tx, nt, groupId); err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

// DeleteGroup deletes a given group by group ID and user ID
func DeleteGroup(g *Group) error {
	// Delete all the group_targets entries for this group
	err := db.Where(queryWhereGroupID, g.Id).Delete(&GroupTarget{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	// Delete the group itself
	err = db.Delete(g).Error
	if err != nil {
		log.Error(err)
		return err
	}
	return err
}

func insertTargetIntoGroup(tx *gorm.DB, t Target, gid int64) error {
	if _, err := mail.ParseAddress(t.Email); err != nil {
		log.WithFields(logrus.Fields{
			"email": t.Email,
		}).Error("Invalid email")
		return err
	}
	err := tx.Where(t).FirstOrCreate(&t).Error
	if err != nil {
		log.WithFields(logrus.Fields{
			"email": t.Email,
		}).Error(err)
		return err
	}
	err = tx.Save(&GroupTarget{GroupId: gid, TargetId: t.Id}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// UpdateTarget updates the given target information in the database.
func UpdateTarget(tx *gorm.DB, target Target) error {
	targetInfo := map[string]interface{}{
		"first_name": target.FirstName,
		"last_name":  target.LastName,
		"position":   target.Position,
		"phone":      target.Phone,
	}
	err := tx.Model(&target).Where("id = ?", target.Id).Updates(targetInfo).Error
	if err != nil {
		log.WithFields(logrus.Fields{
			"email": target.Email,
		}).Error("Error updating target information")
	}
	return err
}

// GetTargets performs a many-to-many select to get all the Targets for a Group
func GetTargets(gid int64) ([]Target, error) {
	ts := []Target{}
	err := db.Table("targets").Select("targets.id, targets.email, targets.first_name, targets.last_name, targets.position, targets.phone").Joins("left join group_targets gt ON targets.id = gt.target_id").Where("gt.group_id=?", gid).Scan(&ts).Error
	return ts, err
}
