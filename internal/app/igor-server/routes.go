// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"net/http"
	"path/filepath"

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
	hcCb := NewHandlerChain(hlog.NewHandler(logger))
	router.Handle(http.MethodGet, api.CbLocal, hcCb.ApplyTo(handleCbs))
	router.Handle(http.MethodGet, api.CbInfo, hcCb.ApplyTo(getInfo))
	router.Handle(http.MethodGet, api.Public, hcCb.ApplyTo(publicShowHandler))
	router.ServeFiles(api.CbKS+"/*filepath", http.Dir(filepath.Join(igor.TFTPPath, igor.KickstartDir)))
	router.ServeFiles(api.CbScript+"/*filepath", http.Dir(igor.Server.ScriptDir))
}

// applyRoutes initializes all route paths
func applyApiRoutes(router *httprouter.Router) {

	//router.HandlerFunc(http.MethodGet, api.BaseUrl+"/debug/pprof/", pprof.Index)
	//router.HandlerFunc(http.MethodGet, api.BaseUrl+"/debug/pprof/cmdline", pprof.Cmdline)
	//router.HandlerFunc(http.MethodGet, api.BaseUrl+"/debug/pprof/profile", pprof.Profile)
	//router.HandlerFunc(http.MethodGet, api.BaseUrl+"/debug/pprof/symbol", pprof.Symbol)
	//router.HandlerFunc(http.MethodGet, api.BaseUrl+"/debug/pprof/trace", pprof.Trace)
	//router.Handler(http.MethodGet, api.BaseUrl+"/debug/pprof/goroutine", pprof.Handler("goroutine"))
	//router.Handler(http.MethodGet, api.BaseUrl+"/debug/pprof/heap", pprof.Handler("heap"))
	//router.Handler(http.MethodGet, api.BaseUrl+"/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	//router.Handler(http.MethodGet, api.BaseUrl+"/debug/pprof/block", pprof.Handler("block"))

	// Default route chain includes logging and checking content type if body if attached
	hcDefaultChain := NewHandlerChain(hlog.NewHandler(logger))
	hcDefaultChain.Add(zlRequestHandler)
	hcDefaultChain.Add(checkContentType)

	// Routes that don't require authentication
	hcPublicShow := NewHandlerChain()
	hcPublicShow.Extend(hcDefaultChain)
	router.Handle(http.MethodGet, api.Public, hcPublicShow.ApplyTo(publicShowHandler))

	hcSettings := NewHandlerChain()
	hcSettings.Extend(hcDefaultChain)
	router.Handle(http.MethodGet, api.PublicSettings, hcSettings.ApplyTo(settingsHandler))

	// IAuth will be applied to most routes
	hcAuthChain := NewHandlerChain(authnHandler, authzHandler)

	hcConfig := NewHandlerChain()
	hcConfig.Extend(hcDefaultChain)
	hcConfig.Extend(hcAuthChain)
	router.Handle(http.MethodGet, api.Config, hcConfig.ApplyTo(configHandler))

	// handles bare login attempt
	hcLogin := NewHandlerChain()
	hcLogin.Extend(hcDefaultChain)
	router.Handle(http.MethodGet, api.Login, hcLogin.ApplyTo(loginGetHandler))

	// handles a login triggered by another command
	hcLoginPost := NewHandlerChain()
	hcLoginPost.Extend(hcDefaultChain)
	router.Handle(http.MethodPost, api.Login, hcLoginPost.ApplyTo(loginPostHandler))

	hcShow := NewHandlerChain()
	hcShow.Extend(hcDefaultChain)
	hcShow.Extend(hcAuthChain)
	router.Handle(http.MethodGet, api.BaseUrl, hcShow.ApplyTo(showHandler))

	// Create clusters
	hcCreateClusters := NewHandlerChain()
	hcCreateClusters.Extend(hcDefaultChain)
	hcCreateClusters.Add(storeJSONBodyHandler)
	hcCreateClusters.Extend(hcAuthChain)
	hcCreateClusters.Add(validateHostParams) // <-- uses the host validator since this is how hosts are created
	router.Handle(http.MethodPost, api.Clusters, hcCreateClusters.ApplyTo(handleCreateClusters))

	// Read clusters
	hcGetClusters := NewHandlerChain()
	hcGetClusters.Extend(hcDefaultChain)
	hcGetClusters.Extend(hcAuthChain)
	hcGetClusters.Add(validateClusterParams)
	router.Handle(http.MethodGet, api.Clusters, hcGetClusters.ApplyTo(handleReadClusters))

	// Create cluster MOTD
	hcCreateMotd := NewHandlerChain()
	hcCreateMotd.Extend(hcDefaultChain)
	hcCreateMotd.Add(storeJSONBodyHandler)
	hcCreateMotd.Extend(hcAuthChain)
	hcCreateMotd.Add(validateMotdParams)
	router.Handle(http.MethodPatch, api.ClusterMotd, hcCreateMotd.ApplyTo(handleUpdateMotd))

	// Read hosts
	hcReadHosts := NewHandlerChain()
	hcReadHosts.Extend(hcDefaultChain)
	hcReadHosts.Extend(hcAuthChain)
	hcReadHosts.Add(validateHostParams)
	router.Handle(http.MethodGet, api.Hosts, hcReadHosts.ApplyTo(handleReadHosts))

	// Update hosts
	hcUpdateHost := NewHandlerChain()
	hcUpdateHost.Extend(hcDefaultChain)
	hcUpdateHost.Add(storeJSONBodyHandler)
	hcUpdateHost.Extend(hcAuthChain)
	hcUpdateHost.Add(validateHostParams)
	router.Handle(http.MethodPatch, api.HostsName, hcUpdateHost.ApplyTo(handleUpdateHost))

	// Delete hosts
	hcDeleteHost := NewHandlerChain()
	hcDeleteHost.Extend(hcDefaultChain)
	hcDeleteHost.Extend(hcAuthChain)
	hcDeleteHost.Add(validateHostParams)
	router.Handle(http.MethodDelete, api.HostsName, hcDeleteHost.ApplyTo(handleDeleteHosts))

	// Power hosts
	hcPowerHosts := NewHandlerChain()
	hcPowerHosts.Extend(hcDefaultChain)
	hcPowerHosts.Add(storeJSONBodyHandler)
	hcPowerHosts.Extend(hcAuthChain)
	hcPowerHosts.Add(validatePowerParams)
	router.Handle(http.MethodPatch, api.HostsPower, hcPowerHosts.ApplyTo(handlePowerHosts))

	// un/block hosts
	hcBlockHosts := NewHandlerChain()
	hcBlockHosts.Extend(hcDefaultChain)
	hcBlockHosts.Add(storeJSONBodyHandler)
	hcBlockHosts.Extend(hcAuthChain)
	hcBlockHosts.Add(validateBlockParams)
	router.Handle(http.MethodPatch, api.HostsBlock, hcBlockHosts.ApplyTo(handleBlockHosts))

	hcApplHostPolicy := NewHandlerChain()
	hcApplHostPolicy.Extend(hcDefaultChain)
	hcApplHostPolicy.Add(storeJSONBodyHandler)
	hcApplHostPolicy.Extend(hcAuthChain)
	hcApplHostPolicy.Add(validateApplyPolicyParams)
	router.Handle(http.MethodPatch, api.HostApplyPolicy, hcApplHostPolicy.ApplyTo(handleApplyPolicy))

	// Create hostPolicy
	hcCreateHostPolicy := NewHandlerChain()
	hcCreateHostPolicy.Extend(hcDefaultChain)
	hcCreateHostPolicy.Add(storeJSONBodyHandler)
	hcCreateHostPolicy.Extend(hcAuthChain)
	hcCreateHostPolicy.Add(validateHostPolicyParams)
	router.Handle(http.MethodPost, api.HostPolicy, hcCreateHostPolicy.ApplyTo(handleCreateHostPolicy))

	// Read hostPolicies
	hcReadHostPolicy := NewHandlerChain()
	hcReadHostPolicy.Extend(hcDefaultChain)
	hcReadHostPolicy.Extend(hcAuthChain)
	hcReadHostPolicy.Add(validateHostPolicyParams)
	router.Handle(http.MethodGet, api.HostPolicy, hcReadHostPolicy.ApplyTo(handleReadHostPolicies))

	// Update hostpolicy
	hcUpdateHostPolicy := NewHandlerChain()
	hcUpdateHostPolicy.Extend(hcDefaultChain)
	hcUpdateHostPolicy.Add(storeJSONBodyHandler)
	hcUpdateHostPolicy.Extend(hcAuthChain)
	hcUpdateHostPolicy.Add(validateHostPolicyParams)
	router.Handle(http.MethodPatch, api.HostPolicyName, hcUpdateHostPolicy.ApplyTo(handleUpdateHostPolicy))

	// Delete hostpolicy
	hcDeleteHostPolicy := NewHandlerChain()
	hcDeleteHostPolicy.Extend(hcDefaultChain)
	hcDeleteHostPolicy.Extend(hcAuthChain)
	hcDeleteHostPolicy.Add(validateHostPolicyParams)
	router.Handle(http.MethodDelete, api.HostPolicyName, hcDeleteHostPolicy.ApplyTo(handleDeleteHostPolicy))

	// Create reservations
	hcCreateResv := NewHandlerChain()
	hcCreateResv.Extend(hcDefaultChain)
	hcCreateResv.Add(storeJSONBodyHandler)
	hcCreateResv.Extend(hcAuthChain)
	hcCreateResv.Add(validateResvParams)
	router.Handle(http.MethodPost, api.Reservations, hcCreateResv.ApplyTo(handleCreateReservations))

	// Read reservations
	hcReadResv := NewHandlerChain()
	hcReadResv.Extend(hcDefaultChain)
	hcReadResv.Extend(hcAuthChain)
	hcReadResv.Add(validateResvParams)
	router.Handle(http.MethodGet, api.Reservations, hcReadResv.ApplyTo(handleReadReservations))

	// Update reservations
	hcUpdateResv := NewHandlerChain()
	hcUpdateResv.Extend(hcDefaultChain)
	hcUpdateResv.Add(storeJSONBodyHandler)
	hcUpdateResv.Extend(hcAuthChain)
	hcUpdateResv.Add(validateResvParams)
	router.Handle(http.MethodPatch, api.ReservationsName, hcUpdateResv.ApplyTo(handleUpdateReservation))

	// Delete reservations
	hcDeleteResv := NewHandlerChain()
	hcDeleteResv.Extend(hcDefaultChain)
	hcDeleteResv.Extend(hcAuthChain)
	hcDeleteResv.Add(validateResvParams)
	router.Handle(http.MethodDelete, api.ReservationsName, hcDeleteResv.ApplyTo(handleDeleteReservations))

	// Create users
	hcCreateUser := NewHandlerChain()
	hcCreateUser.Extend(hcDefaultChain)
	hcCreateUser.Add(storeJSONBodyHandler)
	hcCreateUser.Extend(hcAuthChain)
	hcCreateUser.Add(validateUserParams)
	router.Handle(http.MethodPost, api.Users, hcCreateUser.ApplyTo(handleCreateUser))

	// Read users
	hcReadUsers := NewHandlerChain()
	hcReadUsers.Extend(hcDefaultChain)
	hcReadUsers.Extend(hcAuthChain)
	hcReadUsers.Add(validateUserParams)
	router.Handle(http.MethodGet, api.Users, hcReadUsers.ApplyTo(handleReadUsers))

	// Update users
	hcUpdateUser := NewHandlerChain()
	hcUpdateUser.Extend(hcDefaultChain)
	hcUpdateUser.Add(storeJSONBodyHandler)
	hcUpdateUser.Extend(hcAuthChain)
	hcUpdateUser.Add(validateUserParams)
	router.Handle(http.MethodPatch, api.UsersName, hcUpdateUser.ApplyTo(handleUpdateUser))

	// Delete users
	hcDeleteUsers := NewHandlerChain()
	hcDeleteUsers.Extend(hcDefaultChain)
	hcDeleteUsers.Extend(hcAuthChain)
	hcDeleteUsers.Add(validateUserParams)
	router.Handle(http.MethodDelete, api.UsersName, hcDeleteUsers.ApplyTo(handleDeleteUser))

	// Do elevate user
	hcElevateUsers := NewHandlerChain()
	hcElevateUsers.Extend(hcDefaultChain)
	hcElevateUsers.Extend(hcAuthChain)
	router.Handle(http.MethodPatch, api.Elevate, hcElevateUsers.ApplyTo(handleElevateUser))

	// Check elevate user
	hcCheckElevateUser := NewHandlerChain()
	hcCheckElevateUser.Extend(hcDefaultChain)
	hcCheckElevateUser.Extend(hcAuthChain)
	router.Handle(http.MethodGet, api.Elevate, hcCheckElevateUser.ApplyTo(handleElevateUserStatus))

	// Cancel elevate user
	hcCancelElevateUser := NewHandlerChain()
	hcCancelElevateUser.Extend(hcDefaultChain)
	hcCancelElevateUser.Extend(hcAuthChain)
	router.Handle(http.MethodDelete, api.Elevate, hcCancelElevateUser.ApplyTo(handleElevateUserCancel))

	// Create group
	hcCreateGroup := NewHandlerChain()
	hcCreateGroup.Extend(hcDefaultChain)
	hcCreateGroup.Add(storeJSONBodyHandler)
	hcCreateGroup.Extend(hcAuthChain)
	hcCreateGroup.Add(validateGroupParams)
	router.Handle(http.MethodPost, api.Groups, hcCreateGroup.ApplyTo(handleCreateGroup))

	// Read groups
	hcReadGroups := NewHandlerChain()
	hcReadGroups.Extend(hcDefaultChain)
	hcReadGroups.Extend(hcAuthChain)
	hcReadGroups.Add(validateGroupParams)
	router.Handle(http.MethodGet, api.Groups, hcReadGroups.ApplyTo(handleReadGroups))

	// Update group
	hcUpdateGroup := NewHandlerChain()
	hcUpdateGroup.Extend(hcDefaultChain)
	hcUpdateGroup.Add(storeJSONBodyHandler)
	hcUpdateGroup.Extend(hcAuthChain)
	hcUpdateGroup.Add(validateGroupParams)
	router.Handle(http.MethodPatch, api.GroupsName, hcUpdateGroup.ApplyTo(handleUpdateGroup))

	// Delete group
	hcDeleteGroup := NewHandlerChain()
	hcDeleteGroup.Extend(hcDefaultChain)
	hcDeleteGroup.Extend(hcAuthChain)
	router.Handle(http.MethodDelete, api.GroupsName, hcDeleteGroup.ApplyTo(handleDeleteGroup))

	// Create profiles
	hcCreateProfiles := NewHandlerChain()
	hcCreateProfiles.Extend(hcDefaultChain)
	hcCreateProfiles.Add(storeJSONBodyHandler)
	hcCreateProfiles.Extend(hcAuthChain)
	hcCreateProfiles.Add(validateProfileParams)
	router.Handle(http.MethodPost, api.Profiles, hcCreateProfiles.ApplyTo(handleCreateProfile))

	// Read profiles
	hcReadProfiles := NewHandlerChain()
	hcReadProfiles.Extend(hcDefaultChain)
	hcReadProfiles.Extend(hcAuthChain)
	hcReadProfiles.Add(validateProfileParams)
	router.Handle(http.MethodGet, api.Profiles, hcReadProfiles.ApplyTo(handleReadProfiles))

	// Update profiles
	hcUpdateProfiles := NewHandlerChain()
	hcUpdateProfiles.Extend(hcDefaultChain)
	hcUpdateProfiles.Add(storeJSONBodyHandler)
	hcUpdateProfiles.Extend(hcAuthChain)
	hcUpdateProfiles.Add(validateProfileParams)
	router.Handle(http.MethodPatch, api.ProfileName, hcUpdateProfiles.ApplyTo(handleUpdateProfile))

	// Delete profiles
	hcDeleteProfiles := NewHandlerChain()
	hcDeleteProfiles.Extend(hcDefaultChain)
	hcDeleteProfiles.Extend(hcAuthChain)
	hcDeleteProfiles.Add(validateProfileParams)
	router.Handle(http.MethodDelete, api.ProfileName, hcDeleteProfiles.ApplyTo(handleDeleteProfile))

	// Register distro boot image files
	hcRegisterDistroFiles := NewHandlerChain()
	hcRegisterDistroFiles.Extend(hcDefaultChain)
	hcRegisterDistroFiles.Extend(hcAuthChain)
	hcRegisterDistroFiles.Add(validateDistroImageParams)
	router.Handle(http.MethodPost, api.ImageRegister, hcRegisterDistroFiles.ApplyTo(handleRegisterDistroImage))

	// Read distro images
	hcReadDistroImages := NewHandlerChain()
	hcReadDistroImages.Extend(hcDefaultChain)
	hcReadDistroImages.Extend(hcAuthChain)
	router.Handle(http.MethodGet, api.Images, hcReadDistroImages.ApplyTo(handleReadDistroImage))

	// Delete distro images
	hcDeleteDistroImages := NewHandlerChain()
	hcDeleteDistroImages.Extend(hcDefaultChain)
	hcDeleteDistroImages.Extend(hcAuthChain)
	router.Handle(http.MethodDelete, api.ImagesName, hcDeleteDistroImages.ApplyTo(handleDeleteDistroImage))

	// Create distros
	hcCreateDistros := NewHandlerChain()
	hcCreateDistros.Extend(hcDefaultChain)
	hcCreateDistros.Extend(hcAuthChain)
	hcCreateDistros.Add(validateDistroParams)
	router.Handle(http.MethodPost, api.Distros, hcCreateDistros.ApplyTo(handleCreateDistro))

	// Read distros
	hcReadDistros := NewHandlerChain()
	hcReadDistros.Extend(hcDefaultChain)
	hcReadDistros.Extend(hcAuthChain)
	hcReadDistros.Add(validateDistroParams)
	router.Handle(http.MethodGet, api.Distros, hcReadDistros.ApplyTo(handleReadDistro))

	// Update distros
	hcUpdateDistros := NewHandlerChain()
	hcUpdateDistros.Extend(hcDefaultChain)
	hcUpdateDistros.Extend(hcAuthChain)
	hcUpdateDistros.Add(validateDistroParams)
	router.Handle(http.MethodPatch, api.DistrosName, hcUpdateDistros.ApplyTo(handleUpdateDistro))

	// Delete distros
	hcDeleteDistros := NewHandlerChain()
	hcDeleteDistros.Extend(hcDefaultChain)
	hcDeleteDistros.Extend(hcAuthChain)
	router.Handle(http.MethodDelete, api.DistrosName, hcDeleteDistros.ApplyTo(handleDeleteDistro))

	// Register kickstart files
	hcRegisterKSFiles := NewHandlerChain()
	hcRegisterKSFiles.Extend(hcDefaultChain)
	hcRegisterKSFiles.Extend(hcAuthChain)
	hcRegisterKSFiles.Add(validateKSParams)
	router.Handle(http.MethodPost, api.KickstartRegister, hcRegisterKSFiles.ApplyTo(handleRegisterKickstart))

	// Read kickstarts
	hcReadKickstart := NewHandlerChain()
	hcReadKickstart.Extend(hcDefaultChain)
	hcReadKickstart.Extend(hcAuthChain)
	router.Handle(http.MethodGet, api.Kickstarts, hcReadKickstart.ApplyTo(handleReadKickstart))

	// Update kickstart
	hcUpdateKickstart := NewHandlerChain()
	hcUpdateKickstart.Extend(hcDefaultChain)
	hcUpdateKickstart.Extend(hcAuthChain)
	hcUpdateKickstart.Add(validateKSParams)
	router.Handle(http.MethodPatch, api.KickstartsName, hcUpdateKickstart.ApplyTo(handleUpdateKickstart))

	// Delete kickstarts
	hcDeleteKickstart := NewHandlerChain()
	hcDeleteKickstart.Extend(hcDefaultChain)
	hcDeleteKickstart.Extend(hcAuthChain)
	router.Handle(http.MethodDelete, api.KickstartsName, hcDeleteKickstart.ApplyTo(handleDeleteKickstart))

	// Run sync command
	hcSync := NewHandlerChain()
	hcSync.Extend(hcDefaultChain)
	hcSync.Extend(hcAuthChain)
	hcSync.Add(validateSyncParams)
	router.Handle(http.MethodGet, api.Sync, hcSync.ApplyTo(syncHandler))

	// Run Token IAuth Secret Reset command
	hcTokenAuthKeyReset := NewHandlerChain()
	hcTokenAuthKeyReset.Extend(hcDefaultChain)
	hcTokenAuthKeyReset.Extend(hcAuthChain)
	router.Handle(http.MethodPut, api.AuthReset, hcTokenAuthKeyReset.ApplyTo(handleResetToken))

	// Run Stats
	hcStats := NewHandlerChain()
	hcStats.Extend(hcDefaultChain)
	hcStats.Extend(hcAuthChain)
	router.Handle(http.MethodGet, api.Stats, hcStats.ApplyTo(statsHandler))
}
