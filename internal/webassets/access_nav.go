package webassets

import (
	"fmt"
	"net/url"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/models"
)

// AccountDestination is the account document open to every authenticated session (ISSUE-125).
const AccountDestination = "/account"

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
	// ISSUE-083: ingestion health targets the visited profile when manageable, otherwise the actor's own profile.
	var health *NavLink
	healthProfile := own
	if visited != nil && admin.AuthorizeAction(actor, admin.ManageProfile, visited) == nil {
		healthProfile = visited
	}
	if healthProfile != nil && admin.AuthorizeAction(actor, admin.ManageProfile, healthProfile) == nil {
		health = &NavLink{fmt.Sprintf("/admin/health?profile=%d", healthProfile.ID), "nav.health", "Health", "HeartPulse", section == "health"}
	}
	nav.ManageCatalog = admin.AuthorizeAction(actor, admin.ManageCatalog, nil) == nil
	nav.AdminBadge = nav.ManageCatalog && visited != nil && visited.UserID == actor.ID
	if admin.AuthorizeAction(actor, admin.ManageAccounts, nil) == nil {
		nav.RootLinks = append(nav.RootLinks, NavLink{"/admin?section=profiles", "ui.manage_profiles", "Manage profiles", "UserRoundCog", section == "profiles" && visited == nil})
		// Root sees Health in the Administration group, above server configuration.
		if health != nil {
			nav.RootLinks = append(nav.RootLinks, *health)
			health = nil
		}
	}
	// Profile owners without root keep Health with their account links.
	if health != nil {
		nav.AccountLinks = append(nav.AccountLinks, *health)
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

// Tabs of the shared account shell (ISSUE-126). One set serves the signed-in user's own
// account ("self") and a root operating another account ("context").
const (
	TabAccount = "account"
	TabProfile = "profile"
	TabKeys    = "keys"
	TabHistory = "history"
	TabRelay   = "relay"
)

// accountTabOrder is the visual order of the tabs.
var accountTabOrder = []struct{ Tab, Label, Fallback string }{
	{TabAccount, "account.tab.account", "Account"},
	{TabProfile, "account.tab.profile", "Profile"},
	{TabKeys, "account.tab.keys", "API keys"},
	{TabHistory, "account.tab.history", "History"},
	{TabRelay, "account.tab.relay", "Relay"},
}

// AccountTabURL returns the address of a tab. In self mode the target is the session's own
// profile, so the address never carries an ID; in context mode it is the administration URL.
func AccountTabURL(self bool, profileID int32, tab string) string {
	if self {
		if tab == TabAccount {
			return AccountDestination
		}
		return AccountDestination + "/" + tab
	}
	switch tab {
	case TabAccount:
		return fmt.Sprintf("/admin?section=account&profile=%d", profileID)
	case TabKeys:
		return fmt.Sprintf("/admin?section=keys&profile=%d", profileID)
	case TabHistory:
		return fmt.Sprintf("/admin?section=import&profile=%d", profileID)
	case TabRelay:
		return fmt.Sprintf("/admin?section=relay&profile=%d", profileID)
	}
	return fmt.Sprintf("/admin?profile=%d", profileID)
}

// TabFromSection maps the legacy /admin?section= values to a tab. It reports false for
// sections that are not tabs (for example an unknown value).
func TabFromSection(section string) (string, bool) {
	switch section {
	case "", "profiles":
		return TabProfile, true
	case "account":
		return TabAccount, true
	case "keys":
		return TabKeys, true
	case "import", "export":
		return TabHistory, true
	case "relay":
		return TabRelay, true
	}
	return "", false
}

// AccountTabs lists the five tabs for a target profile, marking current. The caller decides
// the mode from the session, never from the request.
func AccountTabs(self bool, profileID int32, current string) []NavLink {
	tabs := make([]NavLink, 0, len(accountTabOrder))
	for _, t := range accountTabOrder {
		tabs = append(tabs, NavLink{URL: AccountTabURL(self, profileID, t.Tab), Label: t.Label, Fallback: t.Fallback, Current: t.Tab == current})
	}
	return tabs
}
