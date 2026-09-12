// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gorm.io/gorm"

	"github.com/rs/cors"
	//_ "net/http/pprof"
)

// Global Variables
var (
	wg sync.WaitGroup

	resNotifyChan     = make(chan ResNotifyEvent, 100)
	acctNotifyChan    = make(chan AcctNotifyEvent, 100)
	groupNotifyChan   = make(chan GroupNotifyEvent, 100)
	refreshStatusChan = make(chan struct{}, 250)
	clusterUpdateChan = make(chan struct{})
	shutdownChan      = make(chan struct{})

	// initrdQueue is used to queue and process initrd jobs
	initrdQueue *InitrdJobQueue

	// shutdownGrace bounds the whole shutdown sequence -- both HTTP servers and the background
	// workers together, not each stage separately.
	// A var rather than a const only so tests can shorten it; nothing at runtime reassigns it.
	shutdownGrace    = 20 * time.Second
	shutdownDeadline time.Time
)

// runServer sets up and runs the server processes. It blocks until shutdown.
func runServer() {

	// start reservation manager
	wg.Add(1)
	logger.Info().Msg("starting reservation manager")
	go reservationManager()

	// start maintenance manager if a maintenance period has been specified
	if igor.Maintenance.HostMaintenanceDuration > 0 {
		logger.Info().Msgf("starting maintenance manager; host maintanance interval set to %v minutes",
			igor.Maintenance.HostMaintenanceDuration)
		wg.Add(1)
		go maintenanceManager()
	} else {
		logger.Warn().Msg("maintenance manager is disabled")
	}

	// the notification manager will not run if there is no SMTP server configured
	if len(igor.Email.SmtpServer) > 0 {
		logger.Info().Msg("SMTP server configured; starting notification manager")
		wg.Add(1)
		go notificationManager()
	} else {
		logger.Warn().Msg("notification manager is disabled")
	}

	// the group sync manager will not run if disabled in config
	if igor.Auth.Ldap.Sync.EnableUserSync || igor.Auth.Ldap.Sync.EnableGroupSync {
		logger.Info().Msgf("starting LDAP sync manager; sync types (users=%v, groups=%v)",
			igor.Auth.Ldap.Sync.EnableUserSync,
			igor.Auth.Ldap.Sync.EnableGroupSync)
		wg.Add(1)
		go ldapSyncManager()
	} else {
		logger.Warn().Msg("LDAP sync manager is disabled")
	}

	cert, err := tls.LoadX509KeyPair(igor.Server.CertFile, igor.Server.KeyFile)
	if err != nil {
		exitPrintFatal(err.Error())
	}
	logger.Info().Msgf("loaded TLS cert/key pair (cert=%s, key=%s)", igor.Server.CertFile, igor.Server.KeyFile)

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
		// This gives us configurable control over where the Vue.js server is allowed to be installed.
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
		ErrorLog:  newHttpErrorLog("api server"),
	}

	cbRouter := newRouter()
	applyCbRoutes(cbRouter)

	cbSrv := &http.Server{
		Addr: fmt.Sprintf("%s:%d", igor.Server.CbHost, igor.Server.CbPort),
		//ReadTimeout:  5 * time.Second,
		//WriteTimeout: 15 * time.Second,
		//IdleTimeout:  time.Minute,
		Handler:  cbRouter,
		ErrorLog: newHttpErrorLog("callback server"),
	}
	// add TLS to cb server if configured
	if *igor.Server.CbUseTLS {
		cbSrv.TLSConfig = tlsConfig
	}

	go fillKernelInfoBacklog()

	// Initialize and start the initrd job queue
	initrdQueue = NewInitrdJobQueue()
	logger.Info().Msg("created and start new initrd-info job queue")
	// Start the worker goroutine that processes jobs in the queue
	initrdQueue.Start()
	go initrdQueue.EnqueuePendingJobs()

	// interrupt signal sent from terminal or systemd. SIGKILL is deliberately absent --
	// it cannot be caught or handled, so registering it did nothing but mislead.
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)

	go func() {

		s := <-sigint // we block here waiting for a signal, then when we get it ...

		logger.Info().Msgf("received OS signal: %v", s)
		logger.Info().Msg("gracefully shutting down")

		shutdownDeadline = time.Now().Add(shutdownGrace)
		ctx, cancel := context.WithDeadline(context.Background(), shutdownDeadline)
		defer cancel()

		shutdownServer(ctx, apiSrv, "REST service")
		shutdownServer(ctx, cbSrv, "node callback service")

		close(shutdownChan) // shuts down reservationManager and notificationManager
	}()

	startServer(apiSrv, "REST service", sigint, true)
	startServer(cbSrv, "node callback service", sigint, *igor.Server.CbUseTLS)

	waitForWorkers(&wg, shutdownChan)

	sqlDb, _ := igor.IGormDb.GetDB().DB()
	_ = sqlDb.Close()
	logger.Info().Msg("closed database session")
	logger.Info().Msg("**** IGOR-SERVER SHUTDOWN COMPLETED ... GOOD-BYE. ****")
}

func startServer(srv *http.Server, name string, sigint chan os.Signal, useTLS bool) {
	wg.Add(1)
	go func() {
		if useTLS {
			logger.Info().Msgf("igor-server (%s) is listening on https://%s", name, srv.Addr)
			if err := srv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, context.Canceled) {
				logger.Error().Msgf("an error occurred during %s: %v", name, err)
				sigint <- syscall.SIGTERM
			}
		} else {
			logger.Info().Msgf("igor-server (%s) is listening on http://%s", name, srv.Addr)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, context.Canceled) {
				logger.Error().Msgf("an error occurred during %s: %v", name, err)
				sigint <- syscall.SIGTERM
			}
		}
		wg.Done()
	}()
}

func shutdownServer(ctx context.Context, srv *http.Server, name string) {
	if err := srv.Shutdown(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			logger.Error().Msgf("%s did not close within the %v shutdown grace -- connections are "+
				"still active, most likely a request blocked on the database write lock or an "+
				"external call; forcing close", name, shutdownGrace)
			if closeErr := srv.Close(); closeErr != nil {
				logger.Error().Msgf("error force-closing %s: %v", name, closeErr)
			}
		} else {
			logger.Error().Msgf("error shutting down %s: %v", name, err)
		}
	} else {
		logger.Info().Msgf("%s closed", name)
	}
}

// waitForWorkers is the call that keeps runServer alive for the life of the process. It
// blocks indefinitely while the server is running and only starts timing once shutdown has
// actually been requested.
//
// The WaitGroup and the shutdown channel are parameters rather than the package globals so
// that tests can supply their own.
func waitForWorkers(workers *sync.WaitGroup, shutdownReq <-chan struct{}) {
	grace := shutdownGrace

	done := make(chan struct{})
	go func() {
		workers.Wait()
		close(done)
	}()

	// No timer in this select. Until a signal arrives there is nothing to wait out.
	select {
	case <-done:
		// every worker exited without a shutdown ever being requested
		return
	case <-shutdownReq:
	}

	remaining := time.Until(shutdownDeadline)
	if remaining <= 0 {
		logger.Error().Msgf("the %v shutdown grace was already spent closing the HTTP services -- "+
			"continuing without waiting on background workers", grace)
		return
	}

	select {
	case <-done:
		logger.Info().Msg("all background workers stopped")
	case <-time.After(remaining):
		logger.Error().Msgf("background workers did not stop within the %v shutdown grace -- "+
			"continuing shutdown; a worker is most likely blocked on the database write lock",
			grace)
	}
}

// enqueueInitrdJob is a convenience function to let other files enqueue
// a newly created DistroImage for initrd processing.
func enqueueInitrdJob(image *DistroImage) {
	if initrdQueue == nil {
		logger.Warn().Msg("initrd job queue is not initialized; ignoring new job.")
		return
	}
	logger.Info().Msgf("enqueueInitrdJob: enqueuing image for distro '%s'", image.Name)
	initrdQueue.Enqueue(InitrdJob{Image: image})
}

// fillKernelInfoBacklog scans the database for any DistroImage records that
// have an empty or NULL kernel_info field. For each record, it calls
// parseKernelInfo (using the path in .Kernel) and updates the record with
// the returned kernel info and breed.
func fillKernelInfoBacklog() {
	logger.Info().Msg("starting kernel info backlog fill")

	// Get all images with missing kernel_info (either empty string or NULL).
	var images []DistroImage
	var err error

	var emptyKernelInfo = map[string]interface{}{"kernel_info": []string{""}}
	err = performDbTx(func(tx *gorm.DB) error {
		images, err = dbReadImage(emptyKernelInfo, 0, tx)
		return err
	})

	if err != nil {
		logger.Error().Msgf("failed to query for missing kernel_info: %v", err)
		return
	}

	if len(images) == 0 {
		logger.Info().Msg("no images with missing kernel_info found")
		return
	}

	logger.Info().Msgf("found %d images needing kernel_info backfill", len(images))

	for _, image := range images {
		// parseKernelInfo can call 'file' on image.Kernel and parse the results
		kernelInfo, breed := parseKernelInfo(&image)

		image.KernelInfo = kernelInfo
		image.Breed = breed

		var saveErr error
		lockedDbWrite(func() {
			saveErr = performDbTx(func(tx *gorm.DB) error {
				return tx.Save(&image).Error
			})
		})

		if saveErr != nil {
			logger.Error().Msgf("failed to update kernel_info for image %s: %v", image.ImageID, saveErr)
			continue
		}

		logger.Debug().Msgf("updated image %s with kernel_info='%s', breed='%s'",
			image.ImageID, kernelInfo, breed)
	}

	logger.Info().Msg("kernel info backlog fill complete")
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
			logger.Info().Msg("stopping reservation manager")
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
			logger.Info().Msg("stopping notification manager")
			return
		case acctNotifyMsg := <-acctNotifyChan:
			logger.Debug().Msg("received an account event message")
			// do something with the event
			if err := processAcctNotifyEvent(acctNotifyMsg); err != nil {
				logger.Error().Msgf("%v", err)
			}
		case groupNotifyMsg := <-groupNotifyChan:
			logger.Debug().Msg("received a group event message")
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
			logger.Info().Msg("stopping maintenance manager")
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

// ldapSyncManager uses a configurable timer to fire every given interval. When this happens, the syncLdapUsers()
// function is called. The function uses configured settings to get a list of members for a given group from
// LDAP. It then compares the list of members to Igor's user list. Any group members who do not currently have
// a User profile in Igor will have one created for them. If notifications is enabled, the user will receive
// one to inform them they can use Igor.
func ldapSyncManager() {
	defer wg.Done()
	timer := time.Minute * time.Duration(igor.Auth.Ldap.Sync.SyncFrequency)
	countdown := NewScheduleTimer(timer)
	for {
		select {
		case <-shutdownChan:
			logger.Info().Msg("stopping LDAP sync manager")
			return
		case checkTime := <-countdown.t.C:
			if adErr := syncPreCheck(); adErr != nil {
				logger.Warn().Msgf("%v", adErr)
				continue
			}
			var notices notifyBuffer
			lockedDbWrite(func() {
				logger.Debug().Msgf("doing LDAP sync management - %v", checkTime.Format(time.RFC3339))
				if igor.Auth.Ldap.Sync.EnableUserSync {
					executeLdapUserSync(&notices)
				}
				if igor.Auth.Ldap.Sync.EnableGroupSync {
					executeLdapGroupSync()
				}
			})
			notices.flush()
			countdown.reset()
		}
	}
}
