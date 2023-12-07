// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/cors"
	//_ "net/http/pprof"
)

// Global Variables
var (
	wg sync.WaitGroup
	// buffered notification channel used for sending out emails to users
	resNotifyChan    = make(chan ResNotifyEvent, 100)
	acctNotifyChan   = make(chan AcctNotifyEvent, 100)
	groupNotifyChan  = make(chan GroupNotifyEvent, 100)
	refreshPowerChan = make(chan struct{}, 250)
	shutdownChan     = make(chan struct{})
)

// runServer sets up and runs the server processes. It blocks until shutdown.
func runServer() {

	// start reservation manager
	wg.Add(1)
	go reservationManager()

	// start maintenance manager if a maintenance period has been specified
	if igor.Maintenance.HostMaintenanceDuration > 0 {
		wg.Add(1)
		go maintenanceManager()
	} else {
		logger.Warn().Msg("maintenance manager is disabled")
	}

	// the notification manager will not run if there is no SMTP server configured
	if len(igor.Email.SmtpServer) > 0 {
		wg.Add(1)
		go notificationManager()
	} else {
		logger.Warn().Msg("notification manager is disabled")
	}

	// the group sync manager will not run if disabled in config
	if igor.Auth.Ldap.GroupSync.EnableGroupSync {
		wg.Add(1)
		go groupSyncManager()
	} else {
		logger.Warn().Msg("group sync manager is disabled")
	}

	cert, err := tls.LoadX509KeyPair(igor.Server.CertFile, igor.Server.KeyFile)
	if err != nil {
		exitPrintFatal(err.Error())
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	apiRouter := newRouter()
	apiRouter.HandleOPTIONS = true
	applyApiRoutes(apiRouter)

	allowedOrigins := make([]string, 0, len(igor.Server.AllowedOrigins))
	for _, ao := range igor.Server.AllowedOrigins {
		allowedOrigins = append(allowedOrigins, "https://"+ao)
	}

	corsHandler := cors.New(cors.Options{
		// If AllowedOrigins is "*" then AllowedCredentials option always treated as false (not good).
		// This gives us configurable control over where the VueJS server is allowed to be installed.
		// This shouldn't affect the CLI client which talks directly to the port igor-server is listening
		// to for HTTP connections.
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:     []string{"*"},
		AllowCredentials:   true, // must be enabled for cross-site requests to have login credentials
		OptionsPassthrough: true, // depends on HandleOPTIONS setting of httprouter in routes.go
		MaxAge:             30,
	}).Handler(apiRouter)

	apiSrv := &http.Server{
		Addr: fmt.Sprintf("%s:%d", igor.Server.Host, igor.Server.Port),
		//ReadTimeout:  5 * time.Second,
		//WriteTimeout: 15 * time.Second,
		//IdleTimeout:  time.Minute,
		Handler:   corsHandler,
		TLSConfig: tlsConfig,
	}

	cbRouter := newRouter()
	applyCbRoutes(cbRouter)

	cbSrv := &http.Server{
		Addr: fmt.Sprintf("%s:%d", igor.Server.CbHost, igor.Server.CbPort),
		//ReadTimeout:  5 * time.Second,
		//WriteTimeout: 15 * time.Second,
		//IdleTimeout:  time.Minute,
		Handler: cbRouter,
	}
	// add TLS to cb server if configured
	if *igor.Server.CbUseTLS {
		cbSrv.TLSConfig = tlsConfig
	}

	// interrupt signal sent from terminal or systemd
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	go func() {

		s := <-sigint // we block here waiting for a signal, then when we get it ...

		logger.Info().Msgf("received OS signal: %v", s)
		logger.Info().Msg("gracefully shutting down")

		if apiSrvErr := apiSrv.Shutdown(context.Background()); apiSrvErr != nil {
			// Error from closing listeners, or context timeout:
			logger.Error().Msgf("abby-normal REST service shutdown: %v", apiSrvErr)
		} else {
			logger.Info().Msg("REST service closed")
		}

		if cbSrvErr := cbSrv.Shutdown(context.Background()); cbSrvErr != nil {
			// Error from closing listeners, or context timeout:
			logger.Error().Msgf("abby-normal node callback service shutdown: %v", cbSrvErr)
		} else {
			logger.Info().Msg("node callback service closed")
		}

		close(shutdownChan) // shuts down reservationManager and notificationManager
	}()

	wg.Add(1)
	go func() {
		logger.Info().Msgf("igor-server (REST service) is listening on https://%s", apiSrv.Addr)
		if stopErr := apiSrv.ListenAndServeTLS("", ""); stopErr != nil && stopErr != http.ErrServerClosed && stopErr != context.Canceled {
			logger.Error().Msgf("an error occurred during REST service shutdown: %v", stopErr)
			if igor.Server.Port < 1025 {
				logger.Warn().Msgf("port %d normally requires process to run as root", igor.Server.Port)
			}
			sigint <- syscall.SIGKILL
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		if *igor.Server.CbUseTLS {
			logger.Info().Msgf("igor-server (node callback service) is listening on https://%s", cbSrv.Addr)
			if stopErr := cbSrv.ListenAndServeTLS("", ""); stopErr != nil && stopErr != http.ErrServerClosed && stopErr != context.Canceled {
				logger.Error().Msgf("an error occurred during node callback service shutdown: %v", stopErr)
				if igor.Server.CbPort < 1025 {
					logger.Warn().Msgf("port %d normally requires process to run as root", igor.Server.CbPort)
				}
				sigint <- syscall.SIGKILL
			}
		} else {
			logger.Info().Msgf("igor-server (node callback service) is listening on http://%s", cbSrv.Addr)
			if stopErr := cbSrv.ListenAndServe(); stopErr != nil && stopErr != http.ErrServerClosed && stopErr != context.Canceled {
				logger.Error().Msgf("an error occurred during node callback service shutdown: %v", stopErr)
				if igor.Server.CbPort < 1025 {
					logger.Warn().Msgf("port %d normally requires process to run as root", igor.Server.CbPort)
				}
				sigint <- syscall.SIGKILL
			}
		}

		wg.Done()
	}()

	wg.Wait()

	sqlDb, _ := igor.IGormDb.GetDB().DB()
	_ = sqlDb.Close()
	logger.Info().Msg("closed database session")
	logger.Info().Msg("**** IGOR-SERVER SHUTDOWN COMPLETED ... GOOD-BYE. ****")
}

// reservationManager uses a timer to fire at the top of every wall clock minute. When this happens reservations
// that have reached their expiration time are cleaned up, reservations that are scheduled to begin do so, and
// periodic emails about reservations nearing their end are sent out.
func reservationManager() {
	defer wg.Done()
	countdown := NewScheduleTimer(time.Minute)
	for {
		select {
		case <-shutdownChan:
			logger.Info().Msg("stopping reservation management background worker")
			if !countdown.t.Stop() {
				<-countdown.t.C
			}
			return
		case checkTime := <-countdown.t.C:
			logger.Debug().Msgf("doing reservation management - %v", checkTime.Format(time.RFC3339))
			if err := manageReservations(&checkTime, closeoutReservations); err != nil {
				logger.Error().Msgf("%v", err)
			}
			if err := manageReservations(&checkTime, installReservations); err != nil {
				logger.Error().Msgf("%v", err)
			}
			if err := manageReservations(&checkTime, sendExpirationWarnings); err != nil {
				logger.Error().Msgf("%v", err)
			}
			countdown.reset()
		}
	}
}

// notificationManager handles notification events that happen as a result of user or admin actions that require
// sending emails to affected users.
func notificationManager() {
	defer wg.Done()
	countdown := NewScheduleTimer(time.Minute + (30 * time.Second))
	for {
		select {
		case <-shutdownChan:
			logger.Info().Msg("stopping notification background worker")
			return
		case acctNotifyMsg := <-acctNotifyChan:
			logger.Debug().Msg("received an account event message")
			// do something with the event
			if err := processAcctNotifyEvent(acctNotifyMsg); err != nil {
				logger.Error().Msgf("%v", err)
			}
		case groupNotifyMsg := <-groupNotifyChan:
			logger.Debug().Msg("received an account event message")
			// do something with the event
			if err := processGroupNotifyEvent(groupNotifyMsg); err != nil {
				logger.Error().Msgf("%v", err)
			}
		case resNotifyMsg := <-resNotifyChan:
			logger.Debug().Msg("received a reservation event message")
			// do something with the event
			if err := processResNotifyEvent(resNotifyMsg); err != nil {
				logger.Error().Msgf("%v", err)
			}
		case checkTime := <-countdown.t.C:
			// this case is our interrupt for the countdown timer. It will block until the next
			logger.Debug().Msgf("doing notification management - %v", checkTime.Format(time.RFC3339))
			countdown.reset()
		}
	}
}

// maintenanceManager uses a timer to fire at the top of every wall clock minute. When this happens reservations
// that have reached their expiration time are put into maintenance mode where a function is fired to look for
// expired reservations placed into the maintenance table and perform maintenance actions on those reservations
// manager also looks for reservations in maintenance mode where the timer has ended and will perform actions on
// those reservations to take them out of maintenance mode.
func maintenanceManager() {
	defer wg.Done()
	timer := time.Minute
	countdown := NewScheduleTimer(timer)
	for {
		select {
		case <-shutdownChan:
			logger.Info().Msg("stopping maintenance management background worker")
			if !countdown.t.Stop() {
				<-countdown.t.C
			}
			return
		case checkTime := <-countdown.t.C:
			logger.Debug().Msgf("doing maintenance management - %v", checkTime.Format(time.RFC3339))
			if err := doMaintenance(&checkTime, finishMaintenance); err != nil {
				logger.Error().Msgf("%v", err)
			}
			countdown.reset()
		}
	}
}

// groupSyncManager uses a configurable timer to fire every given interval. When this happens, the syncUsers()
// function is called. The function uses configured settings to get a list of members for a given group from
// LDAP. It then compares the list of members to Igor's user list. Any group members who do not currently have
// a User profile in Igor will have one created for them. If notifications is enabled, the user will receive
// one to inform them they can use Igor.
func groupSyncManager() {
	defer wg.Done()
	timer := time.Minute * time.Duration(igor.Auth.Ldap.GroupSync.SyncFrequency)
	countdown := NewScheduleTimer(timer)
	for {
		select {
		case <-shutdownChan:
			logger.Info().Msg("stopping group sync management background worker")
			if !countdown.t.Stop() {
				<-countdown.t.C
			}
			return
		case checkTime := <-countdown.t.C:
			logger.Debug().Msgf("doing group sync management - %v", checkTime.Format(time.RFC3339))
			syncUsers()
			countdown.reset()
		}
	}
}
