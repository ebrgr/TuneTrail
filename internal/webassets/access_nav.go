package webassets

import (
	"fmt"
	"net/url"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/models"
)

// AccountDestination is the existing account document, pending the ISSUE-125 route.
const AccountDestination = "/admin?section=account"

// NavLink describes a translated destination already authorized by the server.
type NavLink struct {
	URL, Label, Fallback, Icon string
	Current                    bool
}

// NavAccess is the permission-dependent part of the shared page navigation.
type NavAccess struct {
	Login, Account, ProfileSettings *NavLink
	AccountLinks, RootLinks         []NavLink
	SignOut                         *NavLink
	ManageListening, ManageCatalog  bool
	AdminBadge                      bool
}

// AccessNavigation derives links from the same policy used at the destination.
func AccessNavigation(actor *models.User, own, visited *models.ListeningProfile, section string) NavAccess {
	var nav NavAccess
	if admin.AuthorizeAction(actor, admin.ManageAccount, nil) != nil {
		nav.Login = &NavLink{"/admin", "ui.sign_in", "Sign in", "KeyRound", section == "login"}
		return nav
	}
	nav.Account = &NavLink{AccountDestination, "nav.my_account", "My account", "LockKeyhole", section == "account"}
	nav.AccountLinks = append(nav.AccountLinks, *nav.Account)
	// The confirmation page validates any requested return for the current actor.
	nav.SignOut = &NavLink{"/admin/sign-out", "nav.sign_out", "Sign out", "LogOut", section == "sign-out"}
	if admin.AuthorizeAction(actor, admin.ManageProfile, own) == nil {
		nav.AccountLinks = append(nav.AccountLinks, NavLink{fmt.Sprintf("/admin?profile=%d", own.ID), "nav.my_profile", "My profile", "UserRoundCog", section == "profiles" && visited != nil && own.ID == visited.ID})
	}
	if admin.AuthorizeAction(actor, admin.ManageProfile, visited) == nil {
		nav.ManageListening = true
		nav.ProfileSettings = &NavLink{fmt.Sprintf("/admin?profile=%d", visited.ID), "nav.profile_settings", "Profile settings", "UserRoundCog", section == "profiles"}
	}
	nav.ManageCatalog = admin.AuthorizeAction(actor, admin.ManageCatalog, nil) == nil
	nav.AdminBadge = nav.ManageCatalog && visited != nil && visited.UserID == actor.ID
	if admin.AuthorizeAction(actor, admin.ManageAccounts, nil) == nil {
		nav.RootLinks = append(nav.RootLinks, NavLink{"/admin?section=profiles", "ui.manage_profiles", "Manage profiles", "UserRoundCog", section == "profiles" && visited == nil})
	}
	if admin.AuthorizeAction(actor, admin.ManageServer, nil) == nil {
		settings := "/admin/settings"
		if visited != nil {
			settings += fmt.Sprintf("?profile=%d", visited.ID)
		}
		nav.RootLinks = append(nav.RootLinks, NavLink{settings, "ui.server_configuration", "Server configuration", "Server", section == "settings"})
	}
	return nav
}

// LoginFor keeps a return destination only when the page supplies a safe read URL.
func (n *NavAccess) LoginFor(destination string) {
	if n.Login != nil && destination != "" && destination != "/" {
		n.Login.URL += "?next=" + url.QueryEscape(destination)
	}
}

// SignOutFor carries an allowlisted read URL to the confirmation page.
func (n *NavAccess) SignOutFor(destination string) {
	if n.SignOut != nil && destination != "" {
		n.SignOut.URL += "?return=" + url.QueryEscape(destination)
	}
}

// OwnProfile finds the canonical profile associated with the authenticated user.
func OwnProfile(actor *models.User, profiles []models.ListeningProfile) *models.ListeningProfile {
	for i := range profiles {
		if actor != nil && profiles[i].UserID == actor.ID {
			return &profiles[i]
		}
	}
	return nil
}
