// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"errors"
	"fmt"
	"io"
	"log/syslog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"igor2/internal/pkg/common"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gookit/color"
	"github.com/mileusna/useragent"
	zl "github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	glog "gorm.io/gorm/logger"
)

var (
	traceColor    = color.New(color.FgLightCyan)
	debugColor    = color.New(color.FgBlue)
	infoColor     = color.New(color.FgGreen)
	warnColor     = color.New(color.FgYellow, color.OpBold)
	errorColor    = color.New(color.FgRed, color.OpBold)
	fatalColor    = color.New(color.FgLightWhite, color.BgRed)
	panicColor    = color.New(color.FgLightRed)
	unkLevelColor = color.New(color.FgLightMagenta)

	// logger is our zerolog logging instance. Its level is controlled from the server configuration YAML file.
	logger           zl.Logger
	loggerInited     bool
	zlRequestHandler = hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {

		user := "-"
		if username, _, ok := r.BasicAuth(); ok && username != "" {
			user = username
		} else if username = r.URL.User.Username(); username != "" {
			user = username
		} else {
			tokenString, _ := extractToken(r)
			if token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, acquireTokenSecret); err == nil {
				if claims, ok2 := token.Claims.(*MyClaims); ok2 {
					user = claims.Username
				}
			}
		}

		remoteAddr := "-"

		fIPList := r.Header.Get(common.XForwardedFor)
		if len(fIPList) > 0 {
			ips := strings.Split(fIPList, ",")
			remoteAddr = strings.TrimSpace(ips[0])
		} else if r.RemoteAddr != "" {
			remoteAddr = r.RemoteAddr
		}

		reqUrl, _ := url.QueryUnescape(r.URL.RequestURI())

		userAgent := "-"

		if r.UserAgent() != "" {
			if strings.HasPrefix(r.UserAgent(), IgorCliPrefix) {
				userAgent = r.UserAgent()
			} else {
				ua := useragent.Parse(r.UserAgent())
				userAgent = ua.Name + "/" + ua.Version
			}
		}

		refresh := r.Header.Get(common.IgorRefreshHeader)
		if refresh == "" || strings.EqualFold(refresh, "false") {
			hlog.FromRequest(r).Info().
				Msgf("%s %s %s %v \"%s %s %v\" %d %d", user, userAgent, remoteAddr, duration, r.Method, r.Proto, reqUrl, status, size)
		} else {
			hlog.FromRequest(r).Debug().
				Msgf("%s %s %s %v \"%s %s %v\" %d %d", user, userAgent, remoteAddr, duration, r.Method, r.Proto, reqUrl, status, size)
		}
	})

	// gLogger passes our zerolog logger into GORM to be used in place of its default. It uses whatever level
	// logger is set to.
	gLogger = glog.New(
		&logger, // our zerolog IO.writer
		glog.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  glog.Error,  // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,       // Disable color
		},
	)
)

func initLog() {

	if len(igor.Log.Dir) == 0 {
		igor.Log.Dir = "/var/log/igor"
	}

	if len(igor.Log.File) == 0 {
		igor.Log.File = "igor.log"
	}

	if len(igor.Log.Level) == 0 {
		igor.Log.Level = "info"
	}

	if len(igor.Log.Syslog.Network) == 0 {
		igor.Log.Syslog.Network = "none"
	}

	configLogPath := filepath.Join(igor.Log.Dir, igor.Log.File)
	isVarLogAvailable := true
	var logFilePath string
	var logDirCreateErr error

	if _, err := os.Stat(igor.Log.Dir); errors.Is(err, os.ErrNotExist) {
		logDirCreateErr = os.MkdirAll(igor.Log.Dir, 0755)
		if logDirCreateErr != nil {
			isVarLogAvailable = false
		}
	}
	if isVarLogAvailable {
		logFilePath = configLogPath
	} else {
		logFilePath = filepath.Join(igor.IgorHome, igor.Log.File)
	}
	logfile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		exitPrintFatal(fmt.Sprintf("can't create log file at %s - %v\n", logFilePath, err))
	}

	zl.TimeFieldFormat = common.DateTimeLogFormat

	ll := strings.ToLower(igor.Log.Level)
	var logLevel zl.Level

	switch ll {
	case "trace":
		logLevel = zl.TraceLevel
	case "debug":
		logLevel = zl.DebugLevel
	case "warn":
		logLevel = zl.WarnLevel
	default:
		logLevel = zl.InfoLevel
	}

	zl.SetGlobalLevel(logLevel)

	var writers []io.Writer

	// This makes the log file and stdout logging look the same
	consoleLog := newConsoleWriter(logfile, true)
	writers = append(writers, consoleLog)

	// This will write the same output to STDOUT when running in development mode
	if DEVMODE {
		consoleOut := newConsoleWriter(os.Stdout, false)
		writers = append(writers, consoleOut)
	}

	// Add syslog if specified in config
	var syslogLevelWriter zl.LevelWriter
	var syslogErr error
	var syslogConfigMsg = "syslog init: no syslog daemon access configured"

	if igor.Log.Syslog.Network != "none" {

		syslogPriority := syslog.LOG_INFO | syslog.LOG_DAEMON
		syslogTag := "igor-server"
		var syslogWriter *syslog.Writer

		if igor.Log.Syslog.Network == "local" {
			if syslogWriter, syslogErr = syslog.New(syslogPriority, syslogTag); syslogErr == nil {
				syslogConfigMsg = "syslog init: connected to local syslog daemon"
			}
		} else {
			if syslogWriter, syslogErr = syslog.Dial(igor.Log.Syslog.Network, igor.Log.Syslog.Addr, syslogPriority, syslogTag); syslogErr == nil {
				syslogConfigMsg = fmt.Sprintf("syslog init: connected to syslog daemon on %s:%s",
					igor.Log.Syslog.Network, igor.Log.Syslog.Addr)
			}
		}

		if syslogErr == nil {

			syslogLevelWriter = zl.SyslogLevelWriter(&igorSyslogWriter{
				writer: syslogWriter,
			})
			writers = append(writers, syslogLevelWriter)
		}
	}

	multi := zl.MultiLevelWriter(writers...)
	logger = zl.New(multi).With().Timestamp().Logger()

	// Begin logging output
	loggerInited = true
	logger.Info().Msg("**** STARTING IGOR-SERVER  ****")
	if DEVMODE {
		logger.Warn().Msg("Igor is running in DEVELOPMENT mode! Outside service calls/responses will be mocked.")
	}
	logger.Info().Msg(common.GetVersion("igor-server", true))
	logger.Trace().Msg("Trace logging level enabled!")
	logger.Info().Msgf("IGOR_HOME located at %v", os.Getenv("IGOR_HOME"))
	if logDirCreateErr != nil {
		logger.Warn().Msgf("igor init log: can't create igor log dir at %s - %v", igor.Log.Dir, logDirCreateErr)
	}
	if logFilePath != configLogPath {
		logger.Warn().Msgf("specified log file location (%s) not available - logging to : %s", configLogPath, logFilePath)
	} else {
		logger.Info().Msgf("logging to %s", logFilePath)
	}

	if syslogErr != nil {
		logger.Error().Msgf("syslog init error - %v", syslogErr)
	} else {
		logger.Info().Msg(syslogConfigMsg)
	}
}

// exitPrintFatal does some standard formatting and printing of an error condition before failing out the app.
func exitPrintFatal(errMsg string) {
	// print fatal to STDERR
	fmt.Fprintln(os.Stderr, fatalColor.Sprintf("igor-server: "+errMsg))
	if loggerInited {
		logger.Fatal().Msg(errMsg)
		// program exits with code 1
	} else {
		os.Exit(1)
	}
}

func newConsoleWriter(w io.Writer, noColor bool) zl.ConsoleWriter {
	consoleOut := zl.ConsoleWriter{Out: w}
	consoleOut.FormatLevel = func(i interface{}) string {
		var l string
		if ll, ok := i.(string); ok {
			switch ll {
			case "trace":
				l = colorize("TRACE", traceColor, noColor)
			case "debug":
				l = colorize("DEBUG", debugColor, noColor)
			case "info":
				l = colorize(" INFO", infoColor, noColor)
			case "warn":
				l = colorize(" WARN", warnColor, noColor)
			case "error":
				l = colorize("ERROR", errorColor, noColor)
			case "fatal":
				l = colorize("FATAL", fatalColor, noColor)
			case "panic":
				l = colorize("PANIC", panicColor, noColor)
			default:
				l = colorize("  ???", unkLevelColor, noColor)
			}
		}
		return fmt.Sprintf("| %5s |", l)
	}
	consoleOut.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	consoleOut.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("%s:", i)
	}
	consoleOut.FormatFieldValue = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("%s", i))
	}
	consoleOut.FormatTimestamp = func(i interface{}) string {
		return colorize(fmt.Sprintf("%s", i), color.New(color.FgWhite), noColor)
	}

	return consoleOut
}

func colorize(s interface{}, c color.Style, disabled bool) string {
	if disabled {
		return fmt.Sprintf("%s", s)
	}
	return c.Sprintf("%s", s)
}

type igorSyslogWriter struct {
	writer *syslog.Writer
}

func (w *igorSyslogWriter) Write(p []byte) (int, error) {
	return w.writer.Write(p)
}

// Trace doesn't write trace level logging output to syslog
func (w *igorSyslogWriter) Trace(_ string) error {
	return nil
}
func (w *igorSyslogWriter) Debug(m string) error {
	return w.writer.Debug(m)
}
func (w *igorSyslogWriter) Info(m string) error {
	return w.writer.Info(m)
}
func (w *igorSyslogWriter) Warning(m string) error {
	return w.writer.Warning(m)
}
func (w *igorSyslogWriter) Err(m string) error {
	return w.writer.Err(m)
}
func (w *igorSyslogWriter) Emerg(m string) error {
	return w.writer.Emerg(m)
}
func (w *igorSyslogWriter) Crit(m string) error {
	return w.writer.Crit(m)
}
