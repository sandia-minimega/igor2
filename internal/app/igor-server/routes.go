// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"igor2/internal/pkg/api"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog/hlog"
)

// newRouter parses and handles a received request
func newRouter() *httprouter.Router {

	router := &httprouter.Router{
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleMethodNotAllowed: true,
		PanicHandler:           panicHandler,
	}

	return router
}

func applyCbRoutes(router *httprouter.Router) {
	var routes []string
	hcCb := NewHandlerChain(hlog.NewHandler(logger))
	router.Handle(http.MethodGet, api.CbLocal, hcCb.ApplyTo(handleCbs))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.CbLocal))
	router.Handle(http.MethodGet, api.CbInfo, hcCb.ApplyTo(getInfo))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.CbInfo))
	router.Handle(http.MethodGet, api.Public, hcCb.ApplyTo(publicShowHandler))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Public))
	logger.Debug().Msgf("registered node callback routes:\n%s", strings.Join(routes, "\n"))
	router.ServeFiles(api.CbKS+"/*filepath", http.Dir(filepath.Join(igor.TFTPPath, igor.KickstartDir)))
	router.ServeFiles(api.CbScript+"/*filepath", http.Dir(igor.Server.ScriptDir))
}

// applyRoutes initializes all route paths
func applyApiRoutes(router *httprouter.Router) {

	var routes []string

	// Default route chain includes logging and checking content type if body if attached
	hcDefaultChain := NewHandlerChain(hlog.NewHandler(logger))
	hcDefaultChain.Add(zlRequestHandler)
	hcDefaultChain.Add(checkContentType)

	// Routes that don't require authentication
	hcPublicShow := NewHandlerChain()
	hcPublicShow.Extend(hcDefaultChain)
	router.Handle(http.MethodGet, api.Public, hcPublicShow.ApplyTo(publicShowHandler))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Public))

	hcSettings := NewHandlerChain()
	hcSettings.Extend(hcDefaultChain)
	router.Handle(http.MethodGet, api.PublicSettings, hcSettings.ApplyTo(settingsHandler))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.PublicSettings))

	// IAuth will be applied to most routes
	hcAuthChain := NewHandlerChain(authnHandler, authzHandler)

	hcConfig := NewHandlerChain()
	hcConfig.Extend(hcDefaultChain)
	hcConfig.Extend(hcAuthChain)
	router.Handle(http.MethodGet, api.Config, hcConfig.ApplyTo(configHandler))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Config))

	// handles bare login attempt
	hcLogin := NewHandlerChain()
	hcLogin.Extend(hcDefaultChain)
	router.Handle(http.MethodGet, api.Login, hcLogin.ApplyTo(loginGetHandler))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Login))

	// handles a login triggered by another command
	hcLoginPost := NewHandlerChain()
	hcLoginPost.Extend(hcDefaultChain)
	router.Handle(http.MethodPost, api.Login, hcLoginPost.ApplyTo(loginPostHandler))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPost, api.Login))

	hcShow := NewHandlerChain()
	hcShow.Extend(hcDefaultChain)
	hcShow.Extend(hcAuthChain)
	router.Handle(http.MethodGet, api.BaseUrl, hcShow.ApplyTo(showHandler))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.BaseUrl))

	// Create clusters
	hcCreateClusters := NewHandlerChain()
	hcCreateClusters.Extend(hcDefaultChain)
	hcCreateClusters.Add(storeJSONBodyHandler)
	hcCreateClusters.Extend(hcAuthChain)
	hcCreateClusters.Add(validateHostParams) // <-- uses the host validator since this is how hosts are created
	router.Handle(http.MethodPost, api.Clusters, hcCreateClusters.ApplyTo(handleCreateClusters))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPost, api.Clusters))

	// Read clusters
	hcGetClusters := NewHandlerChain()
	hcGetClusters.Extend(hcDefaultChain)
	hcGetClusters.Extend(hcAuthChain)
	hcGetClusters.Add(validateClusterParams)
	router.Handle(http.MethodGet, api.Clusters, hcGetClusters.ApplyTo(handleReadClusters))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Clusters))

	// Create cluster MOTD
	hcCreateMotd := NewHandlerChain()
	hcCreateMotd.Extend(hcDefaultChain)
	hcCreateMotd.Add(storeJSONBodyHandler)
	hcCreateMotd.Extend(hcAuthChain)
	hcCreateMotd.Add(validateMotdParams)
	router.Handle(http.MethodPatch, api.ClusterMotd, hcCreateMotd.ApplyTo(handleUpdateMotd))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.ClusterMotd))

	// Read hosts
	hcReadHosts := NewHandlerChain()
	hcReadHosts.Extend(hcDefaultChain)
	hcReadHosts.Extend(hcAuthChain)
	hcReadHosts.Add(validateHostParams)
	router.Handle(http.MethodGet, api.Hosts, hcReadHosts.ApplyTo(handleReadHosts))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Hosts))

	// Update hosts
	hcUpdateHost := NewHandlerChain()
	hcUpdateHost.Extend(hcDefaultChain)
	hcUpdateHost.Add(storeJSONBodyHandler)
	hcUpdateHost.Extend(hcAuthChain)
	hcUpdateHost.Add(validateHostParams)
	router.Handle(http.MethodPatch, api.HostsName, hcUpdateHost.ApplyTo(handleUpdateHost))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.HostsName))

	// Delete hosts
	hcDeleteHost := NewHandlerChain()
	hcDeleteHost.Extend(hcDefaultChain)
	hcDeleteHost.Extend(hcAuthChain)
	hcDeleteHost.Add(validateHostParams)
	router.Handle(http.MethodDelete, api.HostsName, hcDeleteHost.ApplyTo(handleDeleteHosts))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodDelete, api.HostsName))

	// Power hosts
	hcPowerHosts := NewHandlerChain()
	hcPowerHosts.Extend(hcDefaultChain)
	hcPowerHosts.Add(storeJSONBodyHandler)
	hcPowerHosts.Extend(hcAuthChain)
	hcPowerHosts.Add(validatePowerParams)
	router.Handle(http.MethodPatch, api.HostsPower, hcPowerHosts.ApplyTo(handlePowerHosts))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.HostsPower))

	// un/block hosts
	hcBlockHosts := NewHandlerChain()
	hcBlockHosts.Extend(hcDefaultChain)
	hcBlockHosts.Add(storeJSONBodyHandler)
	hcBlockHosts.Extend(hcAuthChain)
	hcBlockHosts.Add(validateBlockParams)
	router.Handle(http.MethodPatch, api.HostsBlock, hcBlockHosts.ApplyTo(handleBlockHosts))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.HostsBlock))

	hcApplHostPolicy := NewHandlerChain()
	hcApplHostPolicy.Extend(hcDefaultChain)
	hcApplHostPolicy.Add(storeJSONBodyHandler)
	hcApplHostPolicy.Extend(hcAuthChain)
	hcApplHostPolicy.Add(validateApplyPolicyParams)
	router.Handle(http.MethodPatch, api.HostApplyPolicy, hcApplHostPolicy.ApplyTo(handleApplyPolicy))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.HostApplyPolicy))

	// Create hostPolicy
	hcCreateHostPolicy := NewHandlerChain()
	hcCreateHostPolicy.Extend(hcDefaultChain)
	hcCreateHostPolicy.Add(storeJSONBodyHandler)
	hcCreateHostPolicy.Extend(hcAuthChain)
	hcCreateHostPolicy.Add(validateHostPolicyParams)
	router.Handle(http.MethodPost, api.HostPolicy, hcCreateHostPolicy.ApplyTo(handleCreateHostPolicy))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPost, api.HostPolicy))

	// Read hostPolicies
	hcReadHostPolicy := NewHandlerChain()
	hcReadHostPolicy.Extend(hcDefaultChain)
	hcReadHostPolicy.Extend(hcAuthChain)
	hcReadHostPolicy.Add(validateHostPolicyParams)
	router.Handle(http.MethodGet, api.HostPolicy, hcReadHostPolicy.ApplyTo(handleReadHostPolicies))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.HostPolicy))

	// Update host policy
	hcUpdateHostPolicy := NewHandlerChain()
	hcUpdateHostPolicy.Extend(hcDefaultChain)
	hcUpdateHostPolicy.Add(storeJSONBodyHandler)
	hcUpdateHostPolicy.Extend(hcAuthChain)
	hcUpdateHostPolicy.Add(validateHostPolicyParams)
	router.Handle(http.MethodPatch, api.HostPolicyName, hcUpdateHostPolicy.ApplyTo(handleUpdateHostPolicy))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.HostPolicyName))

	// Delete host policy
	hcDeleteHostPolicy := NewHandlerChain()
	hcDeleteHostPolicy.Extend(hcDefaultChain)
	hcDeleteHostPolicy.Extend(hcAuthChain)
	hcDeleteHostPolicy.Add(validateHostPolicyParams)
	router.Handle(http.MethodDelete, api.HostPolicyName, hcDeleteHostPolicy.ApplyTo(handleDeleteHostPolicy))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodDelete, api.HostPolicyName))

	// Create reservations
	hcCreateResv := NewHandlerChain()
	hcCreateResv.Extend(hcDefaultChain)
	hcCreateResv.Add(storeJSONBodyHandler)
	hcCreateResv.Extend(hcAuthChain)
	hcCreateResv.Add(validateResvParams)
	router.Handle(http.MethodPost, api.Reservations, hcCreateResv.ApplyTo(handleCreateReservations))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPost, api.Reservations))

	// Read reservations
	hcReadResv := NewHandlerChain()
	hcReadResv.Extend(hcDefaultChain)
	hcReadResv.Extend(hcAuthChain)
	hcReadResv.Add(validateResvParams)
	router.Handle(http.MethodGet, api.Reservations, hcReadResv.ApplyTo(handleReadReservations))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Reservations))

	// Update reservations
	hcUpdateResv := NewHandlerChain()
	hcUpdateResv.Extend(hcDefaultChain)
	hcUpdateResv.Add(storeJSONBodyHandler)
	hcUpdateResv.Extend(hcAuthChain)
	hcUpdateResv.Add(validateResvParams)
	router.Handle(http.MethodPatch, api.ReservationsName, hcUpdateResv.ApplyTo(handleUpdateReservation))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.ReservationsName))

	// Delete reservations
	hcDeleteResv := NewHandlerChain()
	hcDeleteResv.Extend(hcDefaultChain)
	hcDeleteResv.Extend(hcAuthChain)
	hcDeleteResv.Add(validateResvParams)
	router.Handle(http.MethodDelete, api.ReservationsName, hcDeleteResv.ApplyTo(handleDeleteReservations))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodDelete, api.ReservationsName))

	// Create users
	hcCreateUser := NewHandlerChain()
	hcCreateUser.Extend(hcDefaultChain)
	hcCreateUser.Add(storeJSONBodyHandler)
	hcCreateUser.Extend(hcAuthChain)
	hcCreateUser.Add(validateUserParams)
	router.Handle(http.MethodPost, api.Users, hcCreateUser.ApplyTo(handleCreateUser))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPost, api.Users))

	// Read users
	hcReadUsers := NewHandlerChain()
	hcReadUsers.Extend(hcDefaultChain)
	hcReadUsers.Extend(hcAuthChain)
	hcReadUsers.Add(validateUserParams)
	router.Handle(http.MethodGet, api.Users, hcReadUsers.ApplyTo(handleReadUsers))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Users))

	// Update users
	hcUpdateUser := NewHandlerChain()
	hcUpdateUser.Extend(hcDefaultChain)
	hcUpdateUser.Add(storeJSONBodyHandler)
	hcUpdateUser.Extend(hcAuthChain)
	hcUpdateUser.Add(validateUserParams)
	router.Handle(http.MethodPatch, api.UsersName, hcUpdateUser.ApplyTo(handleUpdateUser))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.UsersName))

	// Delete users
	hcDeleteUsers := NewHandlerChain()
	hcDeleteUsers.Extend(hcDefaultChain)
	hcDeleteUsers.Extend(hcAuthChain)
	hcDeleteUsers.Add(validateUserParams)
	router.Handle(http.MethodDelete, api.UsersName, hcDeleteUsers.ApplyTo(handleDeleteUser))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodDelete, api.UsersName))

	// Do elevate user
	hcElevateUsers := NewHandlerChain()
	hcElevateUsers.Extend(hcDefaultChain)
	hcElevateUsers.Extend(hcAuthChain)
	router.Handle(http.MethodPatch, api.Elevate, hcElevateUsers.ApplyTo(handleElevateUser))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.Elevate))

	// Check elevate user
	hcCheckElevateUser := NewHandlerChain()
	hcCheckElevateUser.Extend(hcDefaultChain)
	hcCheckElevateUser.Extend(hcAuthChain)
	router.Handle(http.MethodGet, api.Elevate, hcCheckElevateUser.ApplyTo(handleElevateUserStatus))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Elevate))

	// Cancel elevate user
	hcCancelElevateUser := NewHandlerChain()
	hcCancelElevateUser.Extend(hcDefaultChain)
	hcCancelElevateUser.Extend(hcAuthChain)
	router.Handle(http.MethodDelete, api.Elevate, hcCancelElevateUser.ApplyTo(handleElevateUserCancel))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodDelete, api.Elevate))

	// Create group
	hcCreateGroup := NewHandlerChain()
	hcCreateGroup.Extend(hcDefaultChain)
	hcCreateGroup.Add(storeJSONBodyHandler)
	hcCreateGroup.Extend(hcAuthChain)
	hcCreateGroup.Add(validateGroupParams)
	router.Handle(http.MethodPost, api.Groups, hcCreateGroup.ApplyTo(handleCreateGroup))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPost, api.Groups))

	// Read groups
	hcReadGroups := NewHandlerChain()
	hcReadGroups.Extend(hcDefaultChain)
	hcReadGroups.Extend(hcAuthChain)
	hcReadGroups.Add(validateGroupParams)
	router.Handle(http.MethodGet, api.Groups, hcReadGroups.ApplyTo(handleReadGroups))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Groups))

	// Update group
	hcUpdateGroup := NewHandlerChain()
	hcUpdateGroup.Extend(hcDefaultChain)
	hcUpdateGroup.Add(storeJSONBodyHandler)
	hcUpdateGroup.Extend(hcAuthChain)
	hcUpdateGroup.Add(validateGroupParams)
	router.Handle(http.MethodPatch, api.GroupsName, hcUpdateGroup.ApplyTo(handleUpdateGroup))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.GroupsName))

	// Delete group
	hcDeleteGroup := NewHandlerChain()
	hcDeleteGroup.Extend(hcDefaultChain)
	hcDeleteGroup.Extend(hcAuthChain)
	router.Handle(http.MethodDelete, api.GroupsName, hcDeleteGroup.ApplyTo(handleDeleteGroup))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodDelete, api.GroupsName))

	// Create profiles
	hcCreateProfiles := NewHandlerChain()
	hcCreateProfiles.Extend(hcDefaultChain)
	hcCreateProfiles.Add(storeJSONBodyHandler)
	hcCreateProfiles.Extend(hcAuthChain)
	hcCreateProfiles.Add(validateProfileParams)
	router.Handle(http.MethodPost, api.Profiles, hcCreateProfiles.ApplyTo(handleCreateProfile))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPost, api.Profiles))

	// Read profiles
	hcReadProfiles := NewHandlerChain()
	hcReadProfiles.Extend(hcDefaultChain)
	hcReadProfiles.Extend(hcAuthChain)
	hcReadProfiles.Add(validateProfileParams)
	router.Handle(http.MethodGet, api.Profiles, hcReadProfiles.ApplyTo(handleReadProfiles))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Profiles))

	// Update profiles
	hcUpdateProfiles := NewHandlerChain()
	hcUpdateProfiles.Extend(hcDefaultChain)
	hcUpdateProfiles.Add(storeJSONBodyHandler)
	hcUpdateProfiles.Extend(hcAuthChain)
	hcUpdateProfiles.Add(validateProfileParams)
	router.Handle(http.MethodPatch, api.ProfileName, hcUpdateProfiles.ApplyTo(handleUpdateProfile))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.ProfileName))

	// Delete profiles
	hcDeleteProfiles := NewHandlerChain()
	hcDeleteProfiles.Extend(hcDefaultChain)
	hcDeleteProfiles.Extend(hcAuthChain)
	hcDeleteProfiles.Add(validateProfileParams)
	router.Handle(http.MethodDelete, api.ProfileName, hcDeleteProfiles.ApplyTo(handleDeleteProfile))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodDelete, api.ProfileName))

	// Register distro boot image files
	hcRegisterDistroFiles := NewHandlerChain()
	hcRegisterDistroFiles.Extend(hcDefaultChain)
	hcRegisterDistroFiles.Extend(hcAuthChain)
	hcRegisterDistroFiles.Add(validateDistroImageParams)
	router.Handle(http.MethodPost, api.ImageRegister, hcRegisterDistroFiles.ApplyTo(handleRegisterDistroImage))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPost, api.ImageRegister))

	// Read distro images
	hcReadDistroImages := NewHandlerChain()
	hcReadDistroImages.Extend(hcDefaultChain)
	hcReadDistroImages.Extend(hcAuthChain)
	router.Handle(http.MethodGet, api.Images, hcReadDistroImages.ApplyTo(handleReadDistroImage))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Images))

	// Delete distro images
	hcDeleteDistroImages := NewHandlerChain()
	hcDeleteDistroImages.Extend(hcDefaultChain)
	hcDeleteDistroImages.Extend(hcAuthChain)
	router.Handle(http.MethodDelete, api.ImagesName, hcDeleteDistroImages.ApplyTo(handleDeleteDistroImage))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodDelete, api.ImagesName))

	// Create distros
	hcCreateDistros := NewHandlerChain()
	hcCreateDistros.Extend(hcDefaultChain)
	hcCreateDistros.Extend(hcAuthChain)
	hcCreateDistros.Add(validateDistroParams)
	router.Handle(http.MethodPost, api.Distros, hcCreateDistros.ApplyTo(handleCreateDistro))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPost, api.Distros))

	// Read distros
	hcReadDistros := NewHandlerChain()
	hcReadDistros.Extend(hcDefaultChain)
	hcReadDistros.Extend(hcAuthChain)
	hcReadDistros.Add(validateDistroParams)
	router.Handle(http.MethodGet, api.Distros, hcReadDistros.ApplyTo(handleReadDistro))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Distros))

	// Update distros
	hcUpdateDistros := NewHandlerChain()
	hcUpdateDistros.Extend(hcDefaultChain)
	hcUpdateDistros.Extend(hcAuthChain)
	hcUpdateDistros.Add(validateDistroParams)
	router.Handle(http.MethodPatch, api.DistrosName, hcUpdateDistros.ApplyTo(handleUpdateDistro))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.DistrosName))

	// Delete distros
	hcDeleteDistros := NewHandlerChain()
	hcDeleteDistros.Extend(hcDefaultChain)
	hcDeleteDistros.Extend(hcAuthChain)
	router.Handle(http.MethodDelete, api.DistrosName, hcDeleteDistros.ApplyTo(handleDeleteDistro))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodDelete, api.DistrosName))

	// Register kickstart files
	hcRegisterKSFiles := NewHandlerChain()
	hcRegisterKSFiles.Extend(hcDefaultChain)
	hcRegisterKSFiles.Extend(hcAuthChain)
	hcRegisterKSFiles.Add(validateKSParams)
	router.Handle(http.MethodPost, api.KickstartRegister, hcRegisterKSFiles.ApplyTo(handleRegisterKickstart))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPost, api.KickstartRegister))

	// Read kickstarts
	hcReadKickstart := NewHandlerChain()
	hcReadKickstart.Extend(hcDefaultChain)
	hcReadKickstart.Extend(hcAuthChain)
	// hcReadKickstart.Add(validateKSParams)
	router.Handle(http.MethodGet, api.Kickstarts, hcReadKickstart.ApplyTo(handleReadKickstart))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Kickstarts))

	// Update kickstart
	hcUpdateKickstart := NewHandlerChain()
	hcUpdateKickstart.Extend(hcDefaultChain)
	hcUpdateKickstart.Extend(hcAuthChain)
	hcUpdateKickstart.Add(validateKSParams)
	router.Handle(http.MethodPatch, api.KickstartsName, hcUpdateKickstart.ApplyTo(handleUpdateKickstart))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPatch, api.KickstartsName))

	// Delete kickstarts
	hcDeleteKickstart := NewHandlerChain()
	hcDeleteKickstart.Extend(hcDefaultChain)
	hcDeleteKickstart.Extend(hcAuthChain)
	router.Handle(http.MethodDelete, api.KickstartsName, hcDeleteKickstart.ApplyTo(handleDeleteKickstart))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodDelete, api.KickstartsName))

	// Run sync command
	hcSync := NewHandlerChain()
	hcSync.Extend(hcDefaultChain)
	hcSync.Extend(hcAuthChain)
	hcSync.Add(validateSyncParams)
	router.Handle(http.MethodGet, api.Sync, hcSync.ApplyTo(syncHandler))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Sync))

	// Run Token IAuth Secret Reset command
	hcTokenAuthKeyReset := NewHandlerChain()
	hcTokenAuthKeyReset.Extend(hcDefaultChain)
	hcTokenAuthKeyReset.Extend(hcAuthChain)
	router.Handle(http.MethodPut, api.AuthReset, hcTokenAuthKeyReset.ApplyTo(handleResetToken))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodPut, api.AuthReset))

	// Run Stats
	hcStats := NewHandlerChain()
	hcStats.Extend(hcDefaultChain)
	hcStats.Extend(hcAuthChain)
	router.Handle(http.MethodGet, api.Stats, hcStats.ApplyTo(statsHandler))
	routes = append(routes, fmt.Sprintf("        -> %s %s", http.MethodGet, api.Stats))

	logger.Debug().Msgf("registered REST API routes:\n%s", strings.Join(routes, "\n"))
}
