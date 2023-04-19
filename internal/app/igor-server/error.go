// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"fmt"
	"igor2/internal/pkg/common"
	"net/http"
	"reflect"
	"time"

	zl "github.com/rs/zerolog"
)

func stdErrorResp(rb common.ResponseBody, status int, actionPrefix string, err error, clog *zl.Logger) {
	rb.SetMessage(err.Error())
	if status >= http.StatusInternalServerError {
		clog.Error().Msgf("%s error - %v", actionPrefix, err)
	} else {
		clog.Warn().Msgf("%s failed - %v", actionPrefix, err)
	}
}

// BadCredentialsError is invoked when either username or password
// is incorrect or unrecognized (bad)
type BadCredentialsError struct {
	msg string
}

func (e *BadCredentialsError) Error() string { return e.msg }

// BadParamTypeError is used when a named parameter is not the correct/expected data type.
type BadParamTypeError struct {
	paramName string
	paramVal  interface{}
	reqType   string
}

// NewBadParamTypeError is used when a named parameter cannot be cast to the expected data type.
func NewBadParamTypeError(paramName string, paramVal interface{}, reqType string) *BadParamTypeError {
	return &BadParamTypeError{
		reqType:   reqType,
		paramName: paramName,
		paramVal:  paramVal,
	}
}

func (e *BadParamTypeError) Error() string {
	return fmt.Sprintf("invalid parameter '%s': must be a %s, got %v", e.paramName, e.reqType, reflect.TypeOf(e.paramVal))
}

type UnknownParamError struct {
	paramName string
	paramVal  interface{}
}

// NewUnknownParamError is used when a named parameter is unrecognized.
func NewUnknownParamError(paramName string, paramVal interface{}) *UnknownParamError {
	return &UnknownParamError{
		paramName: paramName,
		paramVal:  paramVal,
	}
}

func (e *UnknownParamError) Error() string {
	return fmt.Sprintf("unknown parameter '%s' received: %v", e.paramName, e.paramVal)
}

type MissingParamError struct {
	paramName string
}

// NewMissingParamError is used when a required parameter is missing. If paramName is empty, the
// error will report no parameters were found.
func NewMissingParamError(paramName string) *MissingParamError {
	return &MissingParamError{
		paramName: paramName,
	}
}

func (e *MissingParamError) Error() string {
	if e.paramName == "" {
		return "requested operation had no parameters"
	}
	return fmt.Sprintf("required parameter '%s' not found", e.paramName)
}

// FileAlreadyExistsError is invoked when attempting to save
// a file to a path where a file of the same name already exists
type FileAlreadyExistsError struct {
	msg string
}

func (e *FileAlreadyExistsError) Error() string { return e.msg }

type HostPolicyConflictError struct {
	msg              string
	groupConflict    bool
	durationConflict bool
	scheduleConflict bool
	scStart          time.Time
	scEnd            time.Time
	conflictHosts    []Host
}

func (e *HostPolicyConflictError) Error() string {

	relevantHosts := namesOfHosts(e.conflictHosts)

	if e.groupConflict {
		e.msg = fmt.Sprintf("the following hosts are policy-restricted and unavailable to the user: %v", relevantHosts)
	} else if e.durationConflict {
		e.msg = fmt.Sprintf("reservation duration exceeds maximum allowed for the following policy-restricted hosts: %v", relevantHosts)
	} else if e.scheduleConflict {
		e.msg = fmt.Sprintf("the following policy-restricted hosts: %v are unavailable for the proposed duration during the times %v and %v", relevantHosts, e.scStart, e.scEnd)
	} else {
		e.msg = "unknown error has occurred during policy check"
	}
	return e.msg
}
