// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

// Package admin implements the dashboard for admins and moderators to change and control aspects of the room.
// Including aliases, allow/deny list managment, invites and settings of the room.
package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/vcraescu/go-paginator/v2"
	"github.com/vcraescu/go-paginator/v2/adapter"
	"github.com/vcraescu/go-paginator/v2/view"
	"go.mindeco.de/http/render"

	"github.com/ssbc/go-ssb-room/v2/internal/network"
	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/roomstate"
	"github.com/ssbc/go-ssb-room/v2/web"
	weberrors "github.com/ssbc/go-ssb-room/v2/web/errors"
	"github.com/ssbc/go-ssb-room/v2/web/i18n"
	"github.com/ssbc/go-ssb-room/v2/web/router"
)

// HTMLTemplates define the list of files the template system should load.
var HTMLTemplates = []string{
	"admin/dashboard.tmpl",
	"admin/menu.tmpl",

	"admin/settings.tmpl",

	"admin/aliases-revoke-confirm.tmpl",

	"admin/denied-keys.tmpl",
	"admin/denied-keys-remove-confirm.tmpl",

	"admin/invite-list.tmpl",
	"admin/invite-revoke-confirm.tmpl",
	"admin/invite-created.tmpl",

	"admin/notice-edit.tmpl",

	"admin/member.tmpl",
	"admin/member-list.tmpl",
	"admin/members-remove-confirm.tmpl",
	"admin/members-show-password-reset-token.tmpl",
}

// Databases is an option struct that encapsulates the required database services
type Databases struct {
	Aliases       roomdb.AliasesService
	AuthFallback  roomdb.AuthFallbackService
	Config        roomdb.RoomConfig
	DeniedKeys    roomdb.DeniedKeysService
	Invites       roomdb.InvitesService
	Notices       roomdb.NoticesService
	Members       roomdb.MembersService
	PinnedNotices roomdb.PinnedNoticesService
}

// Handler supplies the dashboard pages behind the membership middleware.
func Handler(
	netInfo network.ServerEndpointDetails,
	r *render.Renderer,
	roomState *roomstate.Manager,
	fh *weberrors.FlashHelper,
	locHelper *i18n.Helper,
	dbs Databases,
) http.Handler {
	mux := router.Admin(nil)

	urlTo := web.NewURLTo(router.CompleteApp(), netInfo)

	var dashboardHandler = dashboardHandler{
		r:       r,
		flashes: fh,
		netInfo: netInfo,

		dbs:       dbs,
		roomState: roomState,
	}
	mux.Get(router.AdminDashboard).HandlerFunc(r.HTML("admin/dashboard.tmpl", dashboardHandler.overview))

	var sh = settingsHandler{
		r:     r,
		urlTo: urlTo,
		db:    dbs.Config,
		loc:   locHelper,
	}
	mux.Get(router.AdminSettings).HandlerFunc(r.HTML("admin/settings.tmpl", sh.overview))
	mux.Get(router.AdminSettingsSetPrivacy).HandlerFunc(sh.setPrivacy)
	mux.Get(router.AdminSettingsSetLanguage).HandlerFunc(sh.setLanguage)

	mux.Get(router.AdminMenu).HandlerFunc(r.HTML("admin/menu.tmpl", func(w http.ResponseWriter, req *http.Request) (interface{}, error) {
		return map[string]interface{}{}, nil
	}))

	var ah = aliasesHandler{
		r:       r,
		flashes: fh,

		db: dbs.Aliases,
	}
	mux.Get(router.AdminAliasesRevokeConfirm).HandlerFunc(r.HTML("admin/aliases-revoke-confirm.tmpl", ah.revokeConfirm))
	mux.Get(router.AdminAliasesRevoke).HandlerFunc(ah.revoke)

	var dh = deniedKeysHandler{
		r:       r,
		flashes: fh,

		db: dbs.DeniedKeys,

		roomCfg: dbs.Config,
	}
	mux.Get(router.AdminDeniedKeysOverview).HandlerFunc(r.HTML("admin/denied-keys.tmpl", dh.overview))
	mux.Get(router.AdminDeniedKeysAdd).HandlerFunc(dh.add)
	mux.Get(router.AdminDeniedKeysRemoveConfirm).HandlerFunc(r.HTML("admin/denied-keys-remove-confirm.tmpl", dh.removeConfirm))
	mux.Get(router.AdminDeniedKeysRemove).HandlerFunc(dh.remove)

	var mh = membersHandler{
		r:       r,
		flashes: fh,
		urlTo:   urlTo,
		netInfo: netInfo,

		db: dbs.Members,

		fallbackAuthDB: dbs.AuthFallback,
		roomCfgDB:      dbs.Config,
	}
	mux.Get(router.AdminMemberDetails).HandlerFunc(r.HTML("admin/member.tmpl", mh.details))
	mux.Get(router.AdminMembersOverview).HandlerFunc(r.HTML("admin/member-list.tmpl", mh.overview))
	mux.Get(router.AdminMembersAdd).HandlerFunc(mh.add)
	mux.Get(router.AdminMembersChangeRole).HandlerFunc(mh.changeRole)
	mux.Get(router.AdminMembersRemoveConfirm).HandlerFunc(r.HTML("admin/members-remove-confirm.tmpl", mh.removeConfirm))
	mux.Get(router.AdminMembersRemove).HandlerFunc(mh.remove)
	mux.Get(router.AdminMembersCreateFallbackReset).HandlerFunc(r.HTML("admin/members-show-password-reset-token.tmpl", mh.createPasswordResetToken))

	var ih = invitesHandler{
		r:       r,
		flashes: fh,
		urlTo:   urlTo,

		db:     dbs.Invites,
		config: dbs.Config,
	}

	mux.Get(router.AdminInvitesOverview).HandlerFunc(r.HTML("admin/invite-list.tmpl", ih.overview))
	mux.Get(router.AdminInvitesCreate).HandlerFunc(r.HTML("admin/invite-created.tmpl", ih.create))
	mux.Get(router.AdminInvitesRevokeConfirm).HandlerFunc(r.HTML("admin/invite-revoke-confirm.tmpl", ih.revokeConfirm))
	mux.Get(router.AdminInvitesRevoke).HandlerFunc(ih.revoke)

	var nh = noticeHandler{
		r:       r,
		urlTo:   urlTo,
		flashes: fh,

		noticeDB: dbs.Notices,
		pinnedDB: dbs.PinnedNotices,
		roomCfg:  dbs.Config,
	}
	mux.Get(router.AdminNoticeEdit).HandlerFunc(r.HTML("admin/notice-edit.tmpl", nh.edit))
	mux.Get(router.AdminNoticeDraftTranslation).HandlerFunc(r.HTML("admin/notice-edit.tmpl", nh.draftTranslation))
	mux.Get(router.AdminNoticeAddTranslation).HandlerFunc(nh.addTranslation)
	mux.Get(router.AdminNoticeSave).HandlerFunc(nh.save)

	mux.NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		r.Error(rw, req, http.StatusNotFound, weberrors.PageNotFound{Path: req.URL.Path})
	})

	return customStripPrefix("/admin", mux)
}

// how many elements does a paginated page have by default
const defaultPageSize = 20

// paginate receives the total slice and it's length/count, a URL query for the 'limit' and which 'page'.
//
// The members of the map are:
//
//	Entries: the paginated slice
//	Count: the total number of the whole, unpaginated list
//	FirstInView: a bool thats true if you render the first page
//	LastInView: a bool thats true if you render the last page
//	Paginator and View: helpers for rendering the page accessor (see github.com/vcraescu/go-paginator)
//
// TODO: we could return a struct instead but then need to re-think how we embedd it into all the pages where we need it.
//
//	Maybe renderData["Pages"] = paginatedData
func paginate(total interface{}, count int, qry url.Values) (map[string]interface{}, error) {
	pageSize, err := strconv.Atoi(qry.Get("limit"))
	if err != nil {
		pageSize = defaultPageSize
	}

	page, err := strconv.Atoi(qry.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	paginator := paginator.New(adapter.NewSliceAdapter(total), pageSize)
	paginator.SetPage(page)

	var entries []interface{}
	if err = paginator.Results(&entries); err != nil {
		return nil, fmt.Errorf("paginator failed with %w", err)
	}

	view := view.New(paginator)
	pagesSlice, err := view.Pages()
	if err != nil {
		return nil, fmt.Errorf("paginator view.Pages failed with %w", err)
	}
	if len(pagesSlice) == 0 {
		pagesSlice = []int{1}
	}

	last, err := view.Last()
	if err != nil {
		return nil, fmt.Errorf("paginator view.Last failed with %w", err)
	}

	return map[string]interface{}{
		"Entries":     entries,
		"Count":       count,
		"Paginator":   paginator,
		"View":        view,
		"FirstInView": pagesSlice[0] == 1,
		"LastInView":  pagesSlice[len(pagesSlice)-1] == last,
	}, nil
}

// trim prefix if exists (workaround for named router problem)
func customStripPrefix(prefix string, h http.Handler) http.Handler {
	if prefix == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, prefix)
		rp := strings.TrimPrefix(r.URL.RawPath, prefix)
		if len(p) < len(r.URL.Path) && (r.URL.RawPath == "" || len(rp) < len(r.URL.RawPath)) {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			r2.URL.RawPath = rp
			h.ServeHTTP(w, r2)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}
