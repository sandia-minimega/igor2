// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"bytes"
	"errors"
	"html/template"
	"testing"
	"time"

	"igor2/internal/pkg/common"

	zl "github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glog "gorm.io/gorm/logger"
)

// newTestDbBackend points igor.IGormDb at a fresh, empty in-memory SQLite database
// carrying the full schema, and restores the previous backend when the test ends. The
// DSN names a shared-cache database rather than using ":memory:" so that every pooled
// connection reaches the same one.
func newTestDbBackend(t *testing.T) {
	t.Helper()

	dial := &sqlite.Dialector{
		DriverName: "sqlite3_igor",
		DSN:        "file:" + t.Name() + "?mode=memory&cache=shared",
	}
	db, err := gorm.Open(dial, &gorm.Config{Logger: glog.Discard})
	require.NoError(t, err, "opening in-memory database")

	require.NoError(t, db.AutoMigrate(&Permission{}, &User{}, &Group{}, &Host{}, &HostPolicy{},
		&Cluster{}, &Reservation{}, &Kickstart{}, &Distro{}, &Profile{}, &DistroImage{},
		&HistoryRecord{}, &MaintenanceRes{}), "migrating in-memory database")

	prev := igor.IGormDb
	igor.IGormDb = &GormBackend{Database: db}
	t.Cleanup(func() { igor.IGormDb = prev })
}

// captureLog redirects the package logger into a buffer for the duration of the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	prev := logger
	logger = zl.New(buf).Level(zl.TraceLevel)
	t.Cleanup(func() { logger = prev })
	return buf
}

// testEmailConfig points the mailer at a port nothing listens on, so any unintended send
// fails fast rather than hanging, and restores the real settings afterwards.
func testEmailConfig(t *testing.T) {
	t.Helper()

	prev := igor.Email
	t.Cleanup(func() { igor.Email = prev })

	igor.Email.SmtpServer = "127.0.0.1"
	igor.Email.SmtpPort = 1
	igor.Email.DefaultSuffix = "example.com"
}

// initTestTemplates builds tMap so that a test reaching sendEmail renders a real body
// rather than dereferencing a nil template.
func initTestTemplates(t *testing.T) {
	t.Helper()

	resNotifyOff := false
	igor.Email.ResNotifyOn = &resNotifyOff
	initNotify()
}

// testClusterRefs populates the cluster range that the formatHosts template function
// indexes when rendering any template carrying a res-info block.
func testClusterRefs(t *testing.T) {
	t.Helper()

	r, err := common.NewRange("kn", 1, 10)
	require.NoError(t, err)

	prev := igor.ClusterRefs
	igor.ClusterRefs = []common.Range{*r}
	t.Cleanup(func() { igor.ClusterRefs = prev })
}

// dedupeEmailList runs its input through common.Set, which silently discards the empty
// string and whitespace-only entries. A recipient list can therefore look populated to
// a len() check and still resolve to zero addresses.
func TestDedupeEmailListDropsUnusableAddresses(t *testing.T) {

	assert.Empty(t, dedupeEmailList([]string{""}),
		"a lone empty address should resolve to no recipients")
	assert.Empty(t, dedupeEmailList([]string{"   "}),
		"a whitespace-only address should resolve to no recipients")
	assert.Equal(t, []string{"a@b.com"}, dedupeEmailList([]string{"", "a@b.com", "a@b.com"}),
		"usable addresses should survive alongside dropped ones, deduped")
}

// A user with no email on file (igor-admin is seeded that way) yields a one-element
// recipient list holding the empty string. Before the fix that passed the len() guard,
// was emptied by dedupeEmailList, and reached gomail as a header with no addresses --
// which sends MAIL FROM and then DATA with no RCPT TO in between, drawing a
// "503 5.5.2 Need rcpt command" from the SMTP server.
func TestSendEmailRejectsUnusableRecipients(t *testing.T) {

	testEmailConfig(t)
	tmpl := template.Must(template.New("test").Parse("test body"))

	cases := []struct {
		name    string
		toList  []string
		ccList  []string
		bccList []string
	}{
		{"empty owner address", []string{""}, nil, nil},
		{"whitespace owner address", []string{"  "}, nil, nil},
		{"unusable across all three headers", []string{""}, []string{" "}, []string{""}},
		{"no addresses at all", nil, nil, nil},
	}

	for _, c := range cases {
		err := sendEmail(tmpl, "test subject", c.toList, c.ccList, c.bccList, false, struct{}{})
		var noRcpt *NoEmailRecipientError
		assert.True(t, errors.As(err, &noRcpt),
			"%s: expected NoEmailRecipientError, got %v", c.name, err)
	}
}

// The guard must not swallow a message that does have somewhere to go. This one gets as
// far as the dialer and fails there, which is the correct place to fail.
func TestSendEmailAcceptsUsableRecipient(t *testing.T) {

	testEmailConfig(t)
	tmpl := template.Must(template.New("test").Parse("test body"))

	err := sendEmail(tmpl, "test subject", []string{""}, []string{"someone@example.com"}, nil, false, struct{}{})
	var noRcpt *NoEmailRecipientError
	assert.False(t, errors.As(err, &noRcpt),
		"a usable Cc address should not trip the no-recipient guard, got %v", err)
}

// Template.Execute does not nil-check its receiver, so a notify type with no tMap entry
// used to take the whole server down from the notification manager's goroutine.
func TestSendEmailRejectsNilTemplate(t *testing.T) {

	testEmailConfig(t)

	err := sendEmail(nil, "test subject", []string{"someone@example.com"}, nil, nil, false, struct{}{})

	var missing *MissingEmailTemplateError
	assert.True(t, errors.As(err, &missing), "expected MissingEmailTemplateError, got %v", err)
}

// initNotify built no template for EmailGroupRmvOwner, yet group_update.go dispatches it
// whenever an owner is removed from a group.
func TestGroupRmvOwnerRendersAndSends(t *testing.T) {

	testEmailConfig(t)
	captureLog(t)
	initTestTemplates(t)

	require.NotNil(t, tMap[EmailGroupRmvOwner], "EmailGroupRmvOwner must have a template")

	err := processGroupNotifyEvent(GroupNotifyEvent{
		NotifyEvent: NotifyEvent{Type: EmailGroupRmvOwner, Instance: "igor-test"},
		Group:       &Group{Name: "testgroup"},
		Member:      &User{Name: "demoted", Email: "demoted@example.com"},
	})

	// rendering succeeded and the send reached the dead port -- previously a panic
	assert.ErrorContains(t, err, "connect: connection refused")
}

// The reservation-time templates are registered whatever Email.ResNotifyOn says, and the
// flag is honored at dispatch instead. Registration used to be conditional while
// EmailResStart and EmailResExpire were dispatched unguarded, so a reservation starting
// on a resNotifyOn:false deployment executed a nil template.
func TestResTimeTemplatesRegisteredWhenNotifyOff(t *testing.T) {

	testEmailConfig(t)
	captureLog(t)

	resNotifyOff := false
	igor.Email.ResNotifyOn = &resNotifyOff
	initNotify()

	for _, ty := range []struct {
		name string
		key  int
	}{
		{"EmailResStart", EmailResStart},
		{"EmailResExpire", EmailResExpire},
		{"EmailResWarn", EmailResWarn},
		{"EmailResFinalWarn", EmailResFinalWarn},
	} {
		assert.NotNil(t, tMap[ty.key], "%s must have a template even with resNotifyOn off", ty.name)
	}
}

// With the flag off those types must be filtered at dispatch rather than sent. The guard
// used to test 1200-1299 -- the account block, which never reaches this function -- so it
// never fired for the 1100-block types it names.
func TestResTimeEmailsFilteredWhenNotifyOff(t *testing.T) {

	testEmailConfig(t)
	captureLog(t)

	resNotifyOff := false
	igor.Email.ResNotifyOn = &resNotifyOff
	initNotify()

	for _, ty := range []struct {
		name string
		key  int
	}{
		{"EmailResStart", EmailResStart},
		{"EmailResExpire", EmailResExpire},
		{"EmailResExtend", EmailResExtend},
		{"EmailResWarn", EmailResWarn},
		{"EmailResFinalWarn", EmailResFinalWarn},
	} {
		err := processResNotifyEvent(ResNotifyEvent{
			NotifyEvent: NotifyEvent{Type: ty.key, Instance: "igor-test"},
			Cluster:     "testcluster",
			Res: &Reservation{
				Name:  "testres",
				Group: Group{Name: GroupUserPrefix + "owner"},
				Owner: User{Name: "owner", Email: "owner@example.com"},
			},
		})
		// filtered before any send, so nothing dials
		assert.NoError(t, err, "%s should be filtered when resNotifyOn is off", ty.name)
	}
}

// The same types must still send when the flag is on.
func TestResTimeEmailsSendWhenNotifyOn(t *testing.T) {

	testEmailConfig(t)
	captureLog(t)
	testClusterRefs(t)

	resNotifyOn := true
	igor.Email.ResNotifyOn = &resNotifyOn
	initNotify()

	err := processResNotifyEvent(ResNotifyEvent{
		NotifyEvent: NotifyEvent{Type: EmailResStart, Instance: "igor-test"},
		Cluster:     "testcluster",
		Res: &Reservation{
			Name:  "testres",
			Group: Group{Name: GroupUserPrefix + "owner"},
			Owner: User{Name: "owner", Email: "owner@example.com"},
			Hosts: []Host{{Name: "kn1"}},
			Start: time.Now(),
			End:   time.Now().Add(time.Hour),
		},
	})

	assert.ErrorContains(t, err, "connect: connection refused")
}

// Email.HelpLink is a web address, not a mailbox -- every template renders it as an
// <a href>. It used to be handed to gomail as the recipient of the auto-removal alert
// whenever igor-admin had no address of its own, which is the default state since
// igor-admin is seeded with Email: "". gomail rejects it and the alert is lost.
func TestAcctRemovedIssueWarnsInsteadOfEmailingHelpLink(t *testing.T) {

	newTestDbBackend(t)
	testEmailConfig(t)
	log := captureLog(t)
	igor.Email.HelpLink = "https://wiki.example.com/igor"
	initTestTemplates(t)

	// igor-admin exactly as database.go seeds it
	require.NoError(t, igor.IGormDb.GetDB().Create(&User{Name: IgorAdmin, Email: ""}).Error)

	err := processAcctNotifyEvent(AcctNotifyEvent{
		NotifyEvent: NotifyEvent{Type: EmailAcctRemovedIssue, Instance: "igor-test"},
		User:        &User{Name: "departed", Email: "departed@example.com"},
	})

	// No send is attempted at all, so nothing dials. Were HelpLink still standing in as
	// the recipient this would carry the dial failure -- and against a reachable server,
	// gomail's "invalid address", since it connects before it parses addresses.
	assert.NoError(t, err)

	assert.Contains(t, log.String(), `"level":"warn"`, "the lost alert should be logged as a warning")
	assert.Contains(t, log.String(), "departed", "the warning should name the removed account")
	assert.Contains(t, log.String(), "no email address configured",
		"the warning should say why the alert could not be sent")
	assert.NotContains(t, log.String(), "wiki.example.com", "helpLink should play no part in this path")
}

// The warning path must not swallow the alert when igor-admin does have an address.
func TestAcctRemovedIssueEmailsAdminWhenAddressIsSet(t *testing.T) {

	newTestDbBackend(t)
	testEmailConfig(t)
	captureLog(t)

	initTestTemplates(t)

	require.NoError(t, igor.IGormDb.GetDB().Create(&User{Name: IgorAdmin, Email: "admin@example.com"}).Error)

	err := processAcctNotifyEvent(AcctNotifyEvent{
		NotifyEvent: NotifyEvent{Type: EmailAcctRemovedIssue, Instance: "igor-test"},
		User:        &User{Name: "departed", Email: "departed@example.com"},
	})

	// reaching the dead port proves a send was attempted rather than skipped
	assert.ErrorContains(t, err, "connect: connection refused")
	var noRcpt *NoEmailRecipientError
	assert.False(t, errors.As(err, &noRcpt), "a real admin address should not trip the recipient guard")
}
