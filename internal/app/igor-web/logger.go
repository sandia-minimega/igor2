// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorweb

import (
	"fmt"
	"igor2/internal/pkg/common"
	"io"
	"log/syslog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gookit/color"

	zl "github.com/rs/zerolog"
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
	logger       zl.Logger
	loggerInited bool
)

func initLog() {

	if len(igorweb.Log.Dir) == 0 {
		igorweb.Log.Dir = "/var/log/igor"
	}

	if len(igorweb.Log.File) == 0 {
		igorweb.Log.File = "igorweb.log"
	}

	if len(igorweb.Log.Level) == 0 {
		igorweb.Log.Level = "info"
	}

	configLogPath := filepath.Join(igorweb.Log.Dir, igorweb.Log.File)
	isVarLogAvailable := true
	var logFilePath string

	if _, err := os.Stat(igorweb.Log.Dir); os.IsNotExist(err) {
		createErr := os.MkdirAll(igorweb.Log.Dir, 0755)
		if createErr != nil {
			fmt.Fprintf(os.Stderr, "igorweb init log: can't create log dir at %s - %v\n", igorweb.Log.Dir, createErr)
			isVarLogAvailable = false
		}
	}
	if isVarLogAvailable {
		logFilePath = configLogPath
	} else {
		logFilePath = filepath.Join(igorweb.IgorHome, igorweb.Log.File)
	}
	logfile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		exitPrintFatal(fmt.Sprintf("can't create log file at %s - %v\n", logFilePath, err))
	}

	zl.TimeFieldFormat = common.DateTimeLogFormat

	ll := strings.ToLower(igorweb.Log.Level)
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

	// This will write the same output STDOUT as long as the env config value isn't set (production)
	consoleOut := newConsoleWriter(os.Stdout, false)
	writers = append(writers, consoleOut)

	// Add syslog if specified in config
	var syslogLevelWriter zl.LevelWriter = nil
	var syslogErr error = nil
	var syslogConfigMsg = "Syslog init: no syslog daemon access configured"

	if !(igorweb.Log.Syslog.Network == "" || igorweb.Log.Syslog.Network == "none") {

		syslogPriority := syslog.LOG_INFO | syslog.LOG_DAEMON
		syslogTag := "igor-web"
		var syslogWriter *syslog.Writer

		if igorweb.Log.Syslog.Network == "local" {
			if syslogWriter, syslogErr = syslog.New(syslogPriority, syslogTag); syslogErr == nil {
				syslogConfigMsg = "Syslog init: connected to local syslog daemon"
			}
		} else {
			if syslogWriter, syslogErr = syslog.Dial(igorweb.Log.Syslog.Network, igorweb.Log.Syslog.Addr, syslogPriority, syslogTag); syslogErr == nil {
				syslogConfigMsg = fmt.Sprintf("Syslog init: connected to syslog daemon on %s:%s",
					igorweb.Log.Syslog.Network, igorweb.Log.Syslog.Addr)
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
	logger.Info().Msg("**** STARTING IGOR-WEB  ****")
	logger.Info().Msg(common.GetVersion("igor-web-server", true))
	logger.Trace().Msg("Trace logging level enabled!")
	logger.Info().Msgf("IGOR_HOME located at %v", os.Getenv("IGOR_HOME"))
	logger.Info().Msgf("Log level: %s", zl.GlobalLevel().String())
	if logfile.Name() != configLogPath {
		logger.Warn().Msgf("Logging to %s - config log file location not available", logfile.Name())
	} else {
		logger.Info().Msgf("Logging to %s", logfile.Name())
	}
	if syslogErr != nil {
		logger.Error().Msg(syslogErr.Error() + ": syslog access is not available")
	} else {
		logger.Info().Msg(syslogConfigMsg)
	}
}

// exitPrintFatal does some standard formatting and printing of an error condition before failing out the app.
func exitPrintFatal(errMsg string) {
	// print fatal to STDERR
	fmt.Fprintln(os.Stderr, fatalColor.Sprintf("igor-web: "+errMsg))
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
