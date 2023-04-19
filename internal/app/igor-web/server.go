// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorweb

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func Execute(configFilepath *string) {

	igorweb.Started = time.Now()

	if igorweb.IgorHome = os.Getenv("IGOR_HOME"); strings.TrimSpace(igorweb.IgorHome) == "" {
		exitPrintFatal("Environment variable IGOR_HOME not defined")
	}

	initConfig(*configFilepath)
	initLog()
	initConfigCheck()

	// This call will not return until the server terminates
	err := runServer()

	if err != nil {
		if err != context.Canceled {
			exitPrintFatal(fmt.Sprintf("An error occurred: %v", err))
		}
		os.Exit(1)
	}
}

// NewServer initializes the server instance and configures it.
func runServer() error {

	fsHandler := http.FileServer(&spaFileSystem{http.Dir(igorweb.WebServer.FileDir)})
	http.Handle("/", fsHandler)

	cert, err := tls.LoadX509KeyPair(igorweb.WebServer.CertFile, igorweb.WebServer.KeyFile)
	if err != nil {
		exitPrintFatal(err.Error())
	}

	webSrv := &http.Server{
		Addr: fmt.Sprintf("%s:%d", igorweb.WebServer.Host, igorweb.WebServer.Port),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		},
	}
	logger.Info().Msgf("starting igorweb server (%s)", webSrv.Addr)

	// This method is called during server shutdown so we can do other things
	webSrv.RegisterOnShutdown(func() {
		logger.Info().Msg("gracefully shutting down igorweb server")
	})

	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, syscall.SIGINT)
		// sigterm signal sent from systemd
		signal.Notify(sigint, syscall.SIGTERM)

		<-sigint // we block here waiting for a signal, then when we get it ...

		if err := webSrv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			logger.Error().Msgf("Abby-normal igorweb server shutdown: %v\n", err)
		}
	}()

	if err := webSrv.ListenAndServeTLS(igorweb.WebServer.CertFile, igorweb.WebServer.KeyFile); err != http.ErrServerClosed {
		// Error starting or closing listener:
		logger.Error().Msgf("igorweb error: %v", err)
		return err
	}

	logger.Info().Msg("**** IGOR-WEB SHUTDOWN COMPLETED ... GOOD-BYE. ****")
	return nil
}

type spaFileSystem struct {
	root http.FileSystem
}

func (fs *spaFileSystem) Open(name string) (f http.File, err error) {
	f, err = fs.root.Open(name)
	if errors.Is(err, os.ErrNotExist) {
		return fs.root.Open("index.html")
	}
	return
}
