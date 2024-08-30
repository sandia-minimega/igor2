// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

// Package api centralizes the igor API into one spot for all code to reference. From this package we can update future
// changes without having to search the server and CLI code for independent URL strings.
package api

const (
	UrlRoot        = "/igor"
	IgorApiVersion = ""
	BaseUrl        = UrlRoot + IgorApiVersion

	AuthReset         = BaseUrl + "/authreset"
	CbLocal           = BaseUrl + "/cb/svc/local"
	CbInfo            = BaseUrl + "/cb/svc/info"
	CbKS              = BaseUrl + "/cb/svc/ks"
	CbScript          = BaseUrl + "/cb/svc/scripts"
	Clusters          = BaseUrl + "/clusters"
	ClusterMotd       = Clusters + "/motd"
	Config            = BaseUrl + "/config"
	Distros           = BaseUrl + "/distros"
	DistrosName       = Distros + "/:distroName"
	Elevate           = BaseUrl + "/elevate"
	Groups            = BaseUrl + "/groups"
	GroupsName        = Groups + "/:groupName"
	Hosts             = BaseUrl + "/hosts"
	HostsName         = Hosts + "/:hostName"
	HostsCtrl         = BaseUrl + "/hosts-ctrl"
	HostsBlock        = HostsCtrl + "/block"
	HostsPower        = HostsCtrl + "/power"
	HostApplyPolicy   = HostsCtrl + "/policy"
	HostPolicy        = BaseUrl + "/hostpolicy"
	HostPolicyName    = HostPolicy + "/:hostpolicyName"
	Images            = BaseUrl + "/images"
	ImagesName        = Images + "/:imageName"
	ImageRegister     = Images + "/register"
	Kickstarts        = BaseUrl + "/kickstart"
	KickstartsName    = Kickstarts + "/:kickstartName"
	KickstartRegister = Kickstarts + "/register"
	Login             = BaseUrl + "/login"
	Profiles          = BaseUrl + "/profiles"
	ProfileName       = Profiles + "/:profileName"
	Public            = BaseUrl + "/public"
	PublicSettings    = Config + "/public"
	Reservations      = BaseUrl + "/reservations"
	ReservationsName  = Reservations + "/:resName"
	Stats             = BaseUrl + "/stats"
	Sync              = BaseUrl + "/sync"
	Users             = BaseUrl + "/users"
	UsersName         = Users + "/:userName"
)
