// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"strings"
	"time"

	"igor2/internal/pkg/common"

	gomail "gopkg.in/mail.v2"
	"gorm.io/gorm"
)

var (
	ResNotifyTimes = make([]time.Duration, 0)
	tFuncs         template.FuncMap
	tMap           map[int]*template.Template
)

func initNotify() {

	if len(igor.Email.SmtpServer) > 0 {

		tFuncs = template.FuncMap{
			"safeText":       safeText,
			"formatDts":      formatDts,
			"formatHosts":    formatHosts,
			"remainingTime":  remainingTime,
			"ifFullName":     ifFullName,
			"passwordLine":   passwordLine,
			"passwordAction": passwordAction,
			"emailOrName":    emailOrName,
			"isAdmin":        isAdmin,
			"resEdit":        resEdit,
			"replaceInfo":    replaceInfo,
			"ownerEmailList": ownerEmailList,
		}

		var t *template.Template
		tMap = make(map[int]*template.Template)

		setCommonInfo := func(t *template.Template) {
			t, _ = t.Parse(ResInfoTemplate)
			t, _ = t.Parse(SenderInfoTemplate)
		}

		t = template.New("EmailAcctCreated")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyAccountCreatedTemplate)
		t, _ = t.Parse(SenderInfoTemplate)
		tMap[EmailAcctCreated] = t

		t = template.New("EmailPasswordReset")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyPassResetTemplate)
		t, _ = t.Parse(SenderInfoTemplate)
		tMap[EmailPasswordReset] = t

		t = template.New("EmailAcctRemovedIssue")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyAcctRemovedIssue)
		t, _ = t.Parse(SenderInfoTemplate)
		tMap[EmailAcctRemovedIssue] = t

		t = template.New("EmailGroupCreated")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyGroupCreateTemplate)
		t, _ = t.Parse(SenderInfoTemplate)
		tMap[EmailGroupCreated] = t

		t = template.New("EmailGroupAddRmvMem")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyGroupAddRemoveTemplate)
		t, _ = t.Parse(SenderInfoTemplate)
		tMap[EmailGroupAddRmvMem] = t

		t = template.New("EmailGroupAddOwner")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyGroupOwnerChangeTemplate)
		t, _ = t.Parse(SenderInfoTemplate)
		tMap[EmailGroupAddOwner] = t

		t = template.New("EmailGroupChangeName")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyGroupNameChangeTemplate)
		t, _ = t.Parse(SenderInfoTemplate)
		tMap[EmailGroupChangeName] = t

		t = template.New("EmailResEdit")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyResEditTemplate)
		setCommonInfo(t)
		tMap[EmailResEdit] = t

		t = template.New("EmailResDrop")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyResDropTemplate)
		setCommonInfo(t)
		tMap[EmailResDrop] = t

		t = template.New("EmailResBlock")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyResBlockTemplate)
		setCommonInfo(t)
		tMap[EmailResBlock] = t

		t = template.New("EmailResNewOwner")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyResOwnerChangeTemplate)
		setCommonInfo(t)
		tMap[EmailResNewOwner] = t

		t = template.New("EmailResNewGroup")
		t.Funcs(tFuncs)
		t = template.Must(t.Parse(BaseEmailTemplate))
		t, _ = t.Parse(NotifyResGroupChangeTemplate)
		setCommonInfo(t)
		tMap[EmailResNewGroup] = t

		// if reservation notification is turned on, load these
		if *igor.Email.ResNotifyOn {

			t = template.New("EmailResExpire")
			t.Funcs(tFuncs)
			t = template.Must(t.Parse(BaseEmailTemplate))
			t, _ = t.Parse(NotifyResExpireTemplate)
			setCommonInfo(t)
			tMap[EmailResExpire] = t

			t = template.New("EmailResWarn")
			t.Funcs(tFuncs)
			t = template.Must(t.Parse(BaseEmailTemplate))
			t, _ = t.Parse(NotifyResWarnTemplate)
			setCommonInfo(t)
			tMap[EmailResWarn] = t

			t = template.New("EmailResStart")
			t.Funcs(tFuncs)
			t = template.Must(t.Parse(BaseEmailTemplate))
			t, _ = t.Parse(NotifyResStartTemplate)
			setCommonInfo(t)
			tMap[EmailResStart] = t

			t = template.New("EmailResFinalWarn")
			t.Funcs(tFuncs)
			t = template.Must(t.Parse(BaseEmailTemplate))
			t, _ = t.Parse(NotifyResFinalWarnTemplate)
			setCommonInfo(t)
			tMap[EmailResFinalWarn] = t
		}
	}
}

func safeText(s string) template.HTML { return template.HTML(s) }

func formatDts(dts time.Time) string {
	return dts.Format(common.DateTimeEmailFormat)
}

func formatHosts(hosts []Host) string {
	hostnames := namesOfHosts(hosts)
	hostRange, _ := igor.ClusterRefs[0].UnsplitRange(hostnames)
	return hostRange
}

func remainingTime(end time.Time) string {

	timeLeft := time.Until(end).Round(time.Hour)
	hours := int(timeLeft.Hours())
	rDays := hours / 24
	rHours := hours % 24

	daysStr := "days"
	hoursStr := "hours"
	if rDays == 1 {
		daysStr = "day"
	}
	if hours == 1 {
		hoursStr = "hour"
	}

	if rDays > 0 {
		if rHours == 0 {
			return fmt.Sprintf("%d %s", rDays, daysStr)
		}
		return fmt.Sprintf("%d %s and %d %s", rDays, daysStr, rHours, hoursStr)
	}
	return fmt.Sprintf("%d %s", rHours, hoursStr)
}

func ifFullName(name string) string {
	if name != "" {
		return " " + name
	}
	return ""
}

func isAdmin(elevated bool) string {
	if elevated {
		return "an igor administrator"
	}
	return "a reservation group member"
}

func ownerEmailList(owners []User) template.HTML {
	var emails strings.Builder
	for i := 0; i < len(owners); i++ {
		emails.WriteString("<a href=\"mailto:")
		emails.WriteString(owners[i].Email)
		emails.WriteString("\">")
		emails.WriteString(emailOrName(&owners[i]))
		emails.WriteString("</a>")
		if len(owners) > 1 && i < len(owners)-1 {
			emails.WriteString(",")
		}
	}
	return template.HTML(emails.String())
}

func replaceInfo(info string, target string) string {
	if info == "" {
		return target
	}
	return info
}

func resEdit(nType int) string {

	switch nType {
	case EmailResDelete:
		return "deleted"
	case EmailResExtend:
		return "extended"
	case EmailResRename:
		return "renamed"
	default:
		return "edited"
	}
}

func emailOrName(u *User) string {
	if u.Email == "" {
		return "no-reply"
	}
	if u.FullName != "" {
		return u.FullName
	}
	return u.Email
}

func passwordLine(u *User) string {
	if u.Name != IgorAdmin {
		return "Your temporary password is: " + igor.Auth.DefaultUserPassword
	} else {
		return "The igor-admin default password has been restored."
	}
}

func passwordAction(isLocal bool) string {
	if isLocal {
		return "It is recommended that you log in and change your password. " +
			"New igor passwords are 8-16 characters in length and require letters, at least one number, and at least one symbol."
	} else {
		return "You must use your organization's network credentials (LDAP, Kerberos, ...) to log in."
	}
}

type NotifyEvent struct {
	Type     int
	Instance string
	HelpLink string
}

type AcctNotifyEvent struct {
	NotifyEvent
	IsLocal bool
	User    *User
}

// makeAcctNotifyEvent returns a struct to be sent over the 'notify' channel. It returns nil if the email config settings
// prevent email from being sent.
func makeAcctNotifyEvent(nType int, u *User) *AcctNotifyEvent {

	if len(igor.Email.SmtpServer) == 0 {
		logger.Debug().Msgf("no SMTP server defined - user email will not be sent")
		return nil
	}

	authLocal := false
	if igor.Auth.Scheme == "local" {
		authLocal = true
	}

	return &AcctNotifyEvent{
		NotifyEvent: NotifyEvent{
			Type:     nType,
			Instance: igor.InstanceName,
			HelpLink: igor.Email.HelpLink,
		},
		IsLocal: authLocal,
		User:    u,
	}
}

func processAcctNotifyEvent(msg AcctNotifyEvent) error {

	var subj string
	var t *template.Template
	var toList []string
	var ccList []string

	switch msg.Type {

	case EmailAcctCreated:
		subj = "igor account created"
		addEmailToList(&toList, msg.User.Email)
		t = tMap[EmailAcctCreated]
	case EmailPasswordReset:
		subj = "igor account password reset"
		addEmailToList(&toList, msg.User.Email)
		if msg.User.Name == IgorAdmin {
			subj = "igor-admin account password reset"
			queryAdmins := map[string]interface{}{"name": GroupAdmins, "showMembers": true}
			if gList, err := dbReadGroupsTx(queryAdmins, true); err != nil {
				return err
			} else {
				for _, m := range gList[0].Members {
					if m.Name != IgorAdmin {
						addEmailToList(&ccList, m.Email)
					}
				}
			}
		}
		t = tMap[EmailPasswordReset]
	case EmailAcctRemovedIssue:
		subj = "auto-removal of igor account needs review"
		admin, _, _ := getIgorAdminTx()
		if len(admin.Email) != 0 {
			addEmailToList(&toList, admin.Email)
		} else {
			addEmailToList(&toList, igor.Email.HelpLink)
		}
		t = tMap[EmailAcctRemovedIssue]
	default:
		err := fmt.Errorf("unrecognized notify type '%d' - aborting email send", msg.Type)
		logger.Error().Msgf("%v", err)
		return err
	}

	if err := sendEmail(t, subj, toList, ccList, nil, true, msg); err != nil {
		return err
	}

	return nil
}

type GroupNotifyEvent struct {
	NotifyEvent
	Info         string
	Member       *User
	MemberAction string // we fill this is just before invoking template
	Group        *Group
}

// makeGroupNotifyEvent returns a struct to be sent over the notify channel. It returns nil if the email config settings
// prevent email from being sent.
func makeGroupNotifyEvent(nType int, g *Group, m *User, info string) *GroupNotifyEvent {

	if len(igor.Email.SmtpServer) == 0 {
		logger.Debug().Msgf("no SMTP server defined - user email will not be sent")
		return nil
	}

	return &GroupNotifyEvent{
		NotifyEvent: NotifyEvent{
			Type:     nType,
			Instance: igor.InstanceName,
			HelpLink: igor.Email.HelpLink,
		},
		Group:  g,
		Member: m,
		Info:   info,
	}
}

func processGroupNotifyEvent(msg GroupNotifyEvent) error {

	var t *template.Template
	var subj string
	var toList []string
	var ccList []string
	var bccList []string

	switch msg.Type {

	case EmailGroupCreated:
		subj = "new igor group '" + msg.Group.Name + "' created"
		t = tMap[EmailGroupCreated]
		for _, u := range msg.Group.Members {
			addEmailToList(&toList, u.Email)
		}
	case EmailGroupAddMem:
		subj = "igor: you have been added to group '" + msg.Group.Name + "'"
		t = tMap[EmailGroupAddRmvMem]
		addEmailToList(&toList, msg.Member.Email)
		msg.MemberAction = "added to"
	case EmailGroupRmvMem:
		subj = "igor: you have been removed from group '" + msg.Group.Name + "'"
		t = tMap[EmailGroupAddRmvMem]
		addEmailToList(&toList, msg.Member.Email)
		msg.MemberAction = "removed from"
	case EmailGroupAddOwner:
		subj = "igor: you have been added as an owner of group '" + msg.Group.Name + "'"
		t = tMap[EmailGroupAddOwner]
		addEmailToList(&toList, msg.Member.Email)
	case EmailGroupRmvOwner:
		subj = "igor: you have been removed from owner list of group '" + msg.Group.Name + "'"
		t = tMap[EmailGroupRmvOwner]
		addEmailToList(&toList, msg.Member.Email)
	case EmailGroupChangeName:
		subj = "igor: group '" + msg.Info + "' has been renamed"
		t = tMap[EmailGroupChangeName]
		for _, u := range msg.Group.Members {
			addEmailToList(&toList, u.Email)
		}
	default:
		err := fmt.Errorf("unrecognized notify type '%d' - aborting email send", msg.Type)
		logger.Error().Msgf("%v", err)
		return err
	}

	if err := sendEmail(t, subj, toList, ccList, bccList, false, msg); err != nil {
		return err
	}

	return nil
}

type ResNotifyEvent struct {
	NotifyEvent
	Cluster    string
	NextNotify time.Duration
	Res        *Reservation
	ActionUser *User
	IsElevated bool
	Info       string
}

// makeResWarnNotifyEvent returns a struct to be sent over the 'notify' channel. It returns nil if the email config settings
// prevent email from being sent.
func makeResEditNotifyEvent(nType int, r *Reservation, c string, actionUser *User, isElevated bool, info string) *ResNotifyEvent {

	if len(igor.Email.SmtpServer) == 0 {
		logger.Debug().Msgf("no SMTP server defined - user email will not be sent")
		return nil
	}

	return &ResNotifyEvent{
		NotifyEvent: NotifyEvent{
			Type:     nType,
			Instance: igor.InstanceName,
			HelpLink: igor.Email.HelpLink,
		},
		Cluster:    c,
		NextNotify: 0,
		Res:        r,
		ActionUser: actionUser,
		IsElevated: isElevated,
		Info:       info,
	}
}

// makeResWarnNotifyEvent returns a struct to be sent over the 'notify' channel. It returns nil if the email config settings
// prevent email from being sent.
func makeResWarnNotifyEvent(nType int, next time.Duration, r *Reservation, c string) *ResNotifyEvent {

	if len(igor.Email.SmtpServer) == 0 {
		logger.Debug().Msgf("no SMTP server defined - user email will not be sent")
		return nil
	}

	return &ResNotifyEvent{
		NotifyEvent: NotifyEvent{
			Type:     nType,
			Instance: igor.InstanceName,
			HelpLink: igor.Email.HelpLink,
		},
		Cluster:    c,
		NextNotify: next,
		Res:        r,
		ActionUser: nil,
		IsElevated: false,
		Info:       "",
	}
}

func processResNotifyEvent(msg ResNotifyEvent) error {

	// filter out reservation time emails of flag is turned off (extend, expire, time left...)
	if !*igor.Email.ResNotifyOn && 1200 <= msg.Type && msg.Type < 1300 {
		logger.Debug().Msg("reservation time emails are disabled (no email sent)")
		return nil
	}

	var subj string
	var toList []string
	var ccList []string

	var t *template.Template
	priority := false

	subjMid := "'" + msg.Res.Name + "' on " + msg.Cluster

	switch msg.Type {

	case EmailResDelete:
		subj = "igor reservation " + subjMid + " has been deleted"
		t = tMap[EmailResEdit]
		priority = true
	case EmailResDrop:
		subj = "igor reservation " + subjMid + " has dropped host"
		t = tMap[EmailResDrop]
		priority = true
	case EmailResBlock:
		subj = "igor reservation " + subjMid + " has blocked host(s)"
		t = tMap[EmailResBlock]
		priority = true
	case EmailResRename:
		subj = "igor reservation '" + msg.Info + "' on " + msg.Cluster + " has been renamed"
		t = tMap[EmailResEdit]
	case EmailResNewOwner:
		subj = "igor: you are the new owner of reservation " + subjMid
		t = tMap[EmailResNewOwner]
	case EmailResNewGroup:
		subj = "igor reservation " + subjMid + " is now accessible by members of group '" + msg.Res.Group.Name + "'"
		t = tMap[EmailResNewGroup]
	case EmailResExtend:
		subj = "igor reservation " + subjMid + " has been extended"
		t = tMap[EmailResEdit]
	case EmailResExpire:
		subj = "igor reservation " + subjMid + " has expired"
		t = tMap[EmailResExpire]
	case EmailResWarn:
		subj = "igor reservation " + subjMid + " is nearing expiration"
		t = tMap[EmailResWarn]
	case EmailResFinalWarn:
		subj = "FINAL NOTICE: igor reservation " + subjMid + " is expiring soon"
		t = tMap[EmailResFinalWarn]
		priority = true
	case EmailResStart:
		subj = "igor reservation " + subjMid + " has started"
		t = tMap[EmailResStart]
	default:
		err := fmt.Errorf("unrecognized notify type '%d' - aborting email send", msg.Type)
		logger.Error().Msgf("%v", err)
		return err
	}

	if strings.HasPrefix(msg.Res.Group.Name, GroupUserPrefix) {
		toList = append(toList, msg.Res.Owner.Email)
	} else {
		queryParams := map[string]interface{}{"name": msg.Res.Group.Name, "showMembers": true}
		if group, err := dbReadGroupsTx(queryParams, true); err != nil {
			return err
		} else if len(group) > 0 {
			for _, u := range group[0].Members {
				if u.Name == msg.Res.Owner.Name {
					addEmailToList(&toList, u.Email)
				} else if msg.Type != EmailResNewOwner {
					// cc everyone in group except on owner change
					addEmailToList(&ccList, u.Email)
				}
			}
		} else {
			err = fmt.Errorf("unrecognized group name '%s' when trying to notify - no email sent", msg.Res.Group.Name)
			logger.Error().Msgf("%v", err)
			return err
		}
	}

	if err := sendEmail(t, subj, toList, ccList, nil, priority, msg); err != nil {
		return err
	}

	if msg.Type == EmailResWarn || msg.Type == EmailResFinalWarn {

		logger.Info().Msgf("res expire warning sent to members of reservation '%s'", msg.Res.Name)

		dbAccess.Lock()
		defer dbAccess.Unlock()

		if err := performDbTx(func(tx *gorm.DB) error {

			resList, rrErr := dbReadReservations(map[string]interface{}{"name": msg.Res.Name}, nil, tx)
			if rrErr != nil {
				return rrErr
			}
			res := &resList[0]
			changes := map[string]interface{}{"NextNotify": msg.NextNotify}
			return dbEditReservation(res, changes, tx)

		}); err != nil {
			return err
		}
	}

	return nil
}

func addEmailToList(mList *[]string, addr string) {
	if addr != "" {
		*mList = append(*mList, addr)
	}
}

func sendEmail(t *template.Template, subject string, toList []string, ccList []string, bccList []string, isPriority bool, mInfo ...interface{}) error {

	if len(toList) == 0 && len(ccList) == 0 && len(bccList) == 0 {
		return fmt.Errorf("no recipient address for outbound email, subject: %v", subject)
	}
	// Settings for SMTP server
	d := gomail.NewDialer(igor.Email.SmtpServer, igor.Email.SmtpPort, igor.Email.SmtpUsername, igor.Email.SmtpPassword)
	d.RetryFailure = false
	d.TLSConfig = &tls.Config{ServerName: igor.Email.SmtpServer}

	var msgs []*gomail.Message

	for _, info := range mInfo {

		m := gomail.NewMessage()
		m.SetHeader("From", IgorAdmin+"@"+igor.Email.DefaultSuffix)
		if igor.Email.ReplyTo != "" {
			m.SetHeader("Reply-To", igor.Email.ReplyTo)
		}
		m.SetHeader("Subject", subject)
		if len(toList) == 0 && len(ccList) == 0 && len(bccList) == 0 {
			return fmt.Errorf("composed email had no recipients")
		}
		if len(toList) > 0 {
			m.SetHeader("To", toList...)
		}
		if len(ccList) > 0 {
			m.SetHeader("Cc", ccList...)
		}
		if len(bccList) > 0 {
			m.SetHeader("Bcc", bccList...)
		}
		if isPriority {
			m.SetHeader("X-Priority", "1 (Highest)")
			m.SetHeader("X-MSMail-Priority", "High")
			m.SetHeader("Importance", "High")
		}

		var body bytes.Buffer
		if tErr := t.Execute(&body, info); tErr != nil {
			return tErr
		}
		bodyStr := body.String()
		m.SetBody("text/html", bodyStr)
		msgs = append(msgs, m)
	}

	if mailErr := d.DialAndSend(msgs...); mailErr != nil {
		logger.Error().Msgf("%v", mailErr)
		return mailErr
	}
	return nil
}

const (
	EmailResDelete = iota + 1000
	EmailResRename
	EmailResNewOwner
	EmailResNewGroup
	EmailResDrop
	EmailResBlock
	EmailResEdit = 1029
)

const (
	EmailResStart = iota + 1100
	EmailResExtend
	EmailResExpire
	EmailResWarn
	EmailResFinalWarn
)

const (
	EmailAcctCreated = iota + 1200
	EmailPasswordReset
	EmailAcctRemovedIssue
)

const (
	EmailGroupCreated = iota + 1300
	EmailGroupAddMem
	EmailGroupRmvMem
	EmailGroupAddRmvMem
	EmailGroupChangeName
	EmailGroupAddOwner
	EmailGroupRmvOwner
)

const (
	ResInfoTemplate = `
{{template "mail-body" .}}
{{define "res-info"}}
<p>Reservation Name: {{.Res.Name}}
<br>Started: {{formatDts .Res.Start}}
<br>Ends: {{formatDts .Res.End}}
<br>Hosts: {{formatHosts .Res.Hosts}}</p>
{{end}}`

	SenderInfoTemplate = `
{{template "mail-body" .}}
{{define "sender-info"}}
<p>Sincerely,
<br>{{.Instance}}
{{if .HelpLink}}
<br>FAQ/Help: <a href="{{.HelpLink}}">{{.HelpLink}}</a>
{{end}}
{{end}}
`

	NotifyResEditTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>The reservation '{{replaceInfo .Info .Res.Name}}' on the {{.Cluster}} cluster has been {{resEdit .Type}} by <a href="mailto:{{.ActionUser.Email}}">{{emailOrName .ActionUser}}</a>.</p>

<p>This action was undertaken in their role as {{isAdmin .IsElevated}}.</p>

{{block "res-info" .}}{{end}}

{{block "sender-info" .}}{{end}}
{{end}}`

	NotifyResDropTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>The following hosts have been dropped from reservation '{{.Res.Name}}': {{.Info}}</p>

<p>The modified reservation's current info:</p>

{{block "res-info" .}}{{end}}

<p>If you have questions please contact, <a href="mailto:{{.ActionUser.Email}}">{{emailOrName .ActionUser}}</a>. This action was undertaken in their role as {{isAdmin .IsElevated}}.</p>

{{block "sender-info" .}}{{end}}
{{end}}`

	NotifyResBlockTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>The following hosts have been blocked in reservation '{{.Res.Name}}': {{.Info}}</p>

<p>This action is usually undertaken when a cluster admin needs to bring the host(s) offline at some point in the near future to do repairs or upgrades to the hardware. Please reach out to the cluster admin team for more information.</p>

<p>The modified reservation's current info:</p>

{{block "res-info" .}}{{end}}

<p>If you have questions please contact, <a href="mailto:{{.ActionUser.Email}}">{{emailOrName .ActionUser}}</a>. This action was undertaken in their role as {{isAdmin .IsElevated}}.</p>

{{block "sender-info" .}}{{end}}
{{end}}`

	NotifyResOwnerChangeTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>Ownership of the reservation '{{.Res.Name}}' has been transferred to you. If you have questions please contact the former owner, <a href="mailto:{{.ActionUser.Email}}">{{emailOrName .ActionUser}}</a>.

{{block "sender-info" .}}{{end}}
{{end}}
`
	NotifyResGroupChangeTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>The group '{{.Res.Group.Name}}' has been associated with the reservation '{{.Res.Name}}'.

<p>Group membership gives you the ability to send power commands, extend the reservation end time and delete the reservation completely.

{{block "sender-info" .}}{{end}}
{{end}}
`

	NotifyResExpireTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>The following reservation on the {{.Cluster}} cluster has expired:</p>

{{block "res-info" .}}{{end}}

{{block "sender-info" .}}{{end}}
{{end}}`

	NotifyResStartTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>The following reservation was registered on the {{.Cluster}} cluster to start at the date listed below. It is now active.</p>

{{block "res-info" .}}{{end}}

{{block "sender-info" .}}{{end}}
{{end}}`

	NotifyResWarnTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>The following reservation on the {{.Cluster}} cluster has {{remainingTime .Res.End}} left before it expires. You may use the 'extend' command if you wish to continue using this reservation beyond its current end date.</p>

{{block "res-info" .}}{{end}}

{{block "sender-info" .}}{{end}}
{{end}}`

	NotifyResFinalWarnTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>The following reservation on the {{.Cluster}} cluster has {{remainingTime .Res.End}} left before it expires. This is your final notice.</p>

<p>If the administrators have allowed use of the 'extend' command you may be able to continue the reservation beyond its current end date. If you do so a new warning email will be sent at the appropriate time.</p>

{{block "res-info" .}}{{end}}

{{block "sender-info" .}}{{end}}
{{end}}
`

	NotifyAccountCreatedTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings{{ifFullName .User.FullName}},</p>

<p>An igor account has been created for you.</p>

{{if .IsLocal}}
<p>{{passwordLine .User}}</p>
{{end}}

<p>{{passwordAction .IsLocal}}</p>

{{block "sender-info" .}}{{end}}
{{end}}
`

	NotifyPassResetTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings{{ifFullName .User.FullName}},</p>

<p>Your igor account password has been reset by an admin.</p>

<p>{{passwordLine .User}}</p>

<p>{{passwordAction .IsLocal}}</p>

{{block "sender-info" .}}{{end}}
{{end}}
`
	NotifyAcctRemovedIssue = `
{{template "base" .}}
{{define "mail-body"}}
<p>To the Igor administration team,</p>

<p>The account '{{.User.Name}}' has been auto-removed. During this process one or more of the user's groups, reservations and/or distros were re-assigned to igor-admin ownership.</p>

<p>Review these resources and either delete or re-assign their ownership to users they were shared with. Check logs for more information.</p>

{{block "sender-info" .}}{{end}}
{{end}}
`

	NotifyGroupCreateTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>A new group '{{.Group.Name}}' has been created, and you are included as a member. If you have questions please contact the group owner(s), {{ownerEmailList .Group.Owners}}.

<p>Group membership is used to provide access to various igor resources. When applied to a reservation, it gives you the ability to send power commands, extend the reservation end time and delete the reservation completely.

{{block "sender-info" .}}{{end}}
{{end}}
`

	NotifyGroupNameChangeTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>The group '{{.Info}}' has been renamed to '{{.Group.Name}}'. If you have questions please contact the group owner(s), {{ownerEmailList .Group.Owners}}.

{{block "sender-info" .}}{{end}}
{{end}}
`

	NotifyGroupAddRemoveTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>You have been {{.MemberAction}} the group '{{.Group.Name}}'. If you have questions please contact the group owner(s), {{ownerEmailList .Group.Owners}}.

{{block "sender-info" .}}{{end}}
{{end}}
`

	NotifyGroupOwnerChangeTemplate = `
{{template "base" .}}
{{define "mail-body"}}
<p>Greetings,</p>

<p>You have been added as an owner of the group '{{.Group.Name}}'.

{{block "sender-info" .}}{{end}}
{{end}}
`

	BaseEmailTemplate = `
{{define "base"}}
<!DOCTYPE html>
<html lang="en" xmlns="http://www.w3.org/1999/xhtml" xmlns:o="urn:schemas-microsoft-com:office:office">
  <head>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
    <meta name="x-apple-disable-message-reformatting">
    <title></title>
    {{safeText "<!--[if mso]><noscript><xml><o:OfficeDocumentSettings><o:PixelsPerInch>96</o:PixelsPerInch></o:OfficeDocumentSettings></xml></noscript><![endif]-->"}}
  </head>
  <body style="margin:0;padding:0;">
    <table role="presentation" style="width:100%;border-collapse:collapse;border:0;border-spacing:0;background:#ffffff;">
      <tr>
        <td align="left" style="padding:0;">
          {{block "mail-body" .}}{{end}}
        </td>
      </tr>
    </table>
  </body>
</html>
{{end}}
`
)
