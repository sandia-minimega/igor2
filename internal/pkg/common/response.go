// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package common

import "time"

const (
	Success = "success"
	Fail    = "fail"
	Error   = "error"
)

// ResponseBody defines interface accessing the common fields of this type.
type ResponseBody interface {
	SetStatus(httpCode int)
	IsSuccess() bool
	IsFail() bool
	IsError() bool
	SetMessage(msg string)
	GetMessage() string
	GetStatus() string
}

// ResponseBodyBase contains the fields that are common to all structs
// that implement ResponseBody.
type ResponseBodyBase struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	ServerTime string `json:"serverTime"`
}

func NewResponseBodyBase() ResponseBodyBase {
	return ResponseBodyBase{
		ServerTime: time.Now().Local().Format(DateTimeServerFormat),
	}
}

func setStatus(base *ResponseBodyBase, httpCode int) {
	if httpCode >= 200 && httpCode < 300 {
		base.Status = Success
	} else if httpCode >= 400 && httpCode < 500 {
		base.Status = Fail
	} else if httpCode >= 500 {
		base.Status = Error
	} else if httpCode == 302 {
		base.Status = Success
	}
}

func isSuccess(base *ResponseBodyBase) bool {
	return base.Status == Success
}

func isFail(base *ResponseBodyBase) bool {
	return base.Status == Fail
}

func isError(base *ResponseBodyBase) bool {
	return base.Status == Error
}

func setMessage(base *ResponseBodyBase, msg string) {
	base.Message = msg
}

func getMessage(base *ResponseBodyBase) string {
	return base.Message
}

func getStatus(base *ResponseBodyBase) string {
	return base.Status
}

// ResponseBodyBasic casts its Data field as a map[string]interface{}.
type ResponseBodyBasic struct {
	ResponseBodyBase
	Data map[string]interface{} `json:"data"`
}

// NewResponseBody makes a new ResponseBodyBasic.
func NewResponseBody() *ResponseBodyBasic {
	response := &ResponseBodyBasic{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string]interface{}),
	}
	return response
}

func (rb *ResponseBodyBasic) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyBasic) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyBasic) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyBasic) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyBasic) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyBasic) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyBasic) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyClusters casts its Data field as an array of ClusterData.
type ResponseBodyClusters struct {
	ResponseBodyBase
	Data map[string][]ClusterData `json:"data"`
}

func NewResponseBodyClusters() *ResponseBodyClusters {
	response := &ResponseBodyClusters{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string][]ClusterData),
	}
	return response
}

func (rb *ResponseBodyClusters) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyClusters) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyClusters) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyClusters) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyClusters) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyClusters) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyClusters) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyHosts casts its Data field as an array of HostData.
type ResponseBodyHosts struct {
	ResponseBodyBase
	Data map[string][]HostData `json:"data"`
}

func NewResponseBodyHosts() *ResponseBodyHosts {
	response := &ResponseBodyHosts{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string][]HostData),
	}
	return response
}

func (rb *ResponseBodyHosts) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyHosts) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyHosts) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyHosts) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyHosts) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyHosts) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyHosts) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyShow casts its Data field as ShowData
type ResponseBodyShow struct {
	ResponseBodyBase
	Data map[string]ShowData `json:"data"`
}

func NewResponseBodyShow() *ResponseBodyShow {
	response := &ResponseBodyShow{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string]ShowData),
	}
	return response
}

func (rb *ResponseBodyShow) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyShow) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyShow) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyShow) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyShow) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyShow) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyShow) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyUsers casts its Data field as UserData
type ResponseBodyUsers struct {
	ResponseBodyBase
	Data map[string][]UserData `json:"data"`
}

func NewResponseBodyUsers() *ResponseBodyUsers {
	response := &ResponseBodyUsers{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string][]UserData),
	}
	return response
}

func (rb *ResponseBodyUsers) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyUsers) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyUsers) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyUsers) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyUsers) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyUsers) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyUsers) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyGroups casts its Data field as GroupData
type ResponseBodyGroups struct {
	ResponseBodyBase
	Data map[string][]GroupData `json:"data"`
}

func NewResponseBodyGroups() *ResponseBodyGroups {
	response := &ResponseBodyGroups{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string][]GroupData),
	}
	response.Data["owner"] = make([]GroupData, 0)
	response.Data["member"] = make([]GroupData, 0)
	return response
}

func (rb *ResponseBodyGroups) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyGroups) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyGroups) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyGroups) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyGroups) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyGroups) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyGroups) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyPolicies casts its Data field as HostPolicyData
type ResponseBodyPolicies struct {
	ResponseBodyBase
	Data map[string][]HostPolicyData `json:"data"`
}

func NewResponseBodyPolicies() *ResponseBodyPolicies {
	response := &ResponseBodyPolicies{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string][]HostPolicyData),
	}
	return response
}

func (rb *ResponseBodyPolicies) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyPolicies) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyPolicies) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyPolicies) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyPolicies) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyPolicies) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyPolicies) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyImages casts its Data field as DistroData
type ResponseBodyImages struct {
	ResponseBodyBase
	Data map[string][]DistroImageData `json:"data"`
}

func NewResponseBodyImages() *ResponseBodyImages {
	response := &ResponseBodyImages{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string][]DistroImageData),
	}
	return response
}

func (rb *ResponseBodyImages) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyImages) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyImages) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyImages) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyImages) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyImages) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyImages) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyKickstarts casts its Data field as KickstartData
type ResponseBodyKickstarts struct {
	ResponseBodyBase
	Data map[string][]KickstartData `json:"data"`
}

func NewResponseKickstarts() *ResponseBodyKickstarts {
	response := &ResponseBodyKickstarts{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string][]KickstartData),
	}
	return response
}

func (rb *ResponseBodyKickstarts) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyKickstarts) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyKickstarts) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyKickstarts) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyKickstarts) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyKickstarts) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyKickstarts) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyDistros casts its Data field as DistroData
type ResponseBodyDistros struct {
	ResponseBodyBase
	Data map[string][]DistroData `json:"data"`
}

func NewResponseBodyDistros() *ResponseBodyDistros {
	response := &ResponseBodyDistros{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string][]DistroData),
	}
	return response
}

func (rb *ResponseBodyDistros) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyDistros) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyDistros) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyDistros) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyDistros) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyDistros) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyDistros) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyProfiles casts its Data field as ProfileData
type ResponseBodyProfiles struct {
	ResponseBodyBase
	Data map[string][]ProfileData `json:"data"`
}

func NewResponseBodyProfiles() *ResponseBodyProfiles {
	response := &ResponseBodyProfiles{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string][]ProfileData),
	}
	return response
}

func (rb *ResponseBodyProfiles) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyProfiles) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyProfiles) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyProfiles) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyProfiles) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyProfiles) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyProfiles) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyReservations casts its Data field as ReservationData
type ResponseBodyReservations struct {
	ResponseBodyBase
	Data map[string][]ReservationData `json:"data"`
}

func NewResponseBodyReservations() *ResponseBodyReservations {
	response := &ResponseBodyReservations{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string][]ReservationData),
	}
	return response
}

func (rb *ResponseBodyReservations) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyReservations) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyReservations) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyReservations) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyReservations) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyReservations) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyReservations) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodyStats casts its Data field as StatsData
type ResponseBodyStats struct {
	ResponseBodyBase
	Data map[string]StatsData `json:"data"`
}

func NewResponseBodyStats() *ResponseBodyStats {
	response := &ResponseBodyStats{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string]StatsData),
	}
	return response
}

func (rb *ResponseBodyStats) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodyStats) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyStats) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyStats) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyStats) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodyStats) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodyStats) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}

// ResponseBodySync casts its Data field as StatsData
type ResponseBodySync struct {
	ResponseBodyBase
	Data map[string]interface{} `json:"data"`
}

func NewResponseBodySync() *ResponseBodySync {
	response := &ResponseBodySync{
		ResponseBodyBase: NewResponseBodyBase(),
		Data:             make(map[string]interface{}),
	}
	return response
}

func (rb *ResponseBodySync) SetStatus(httpCode int) {
	setStatus(&rb.ResponseBodyBase, httpCode)
}

func (rb *ResponseBodySync) IsSuccess() bool {
	return isSuccess(&rb.ResponseBodyBase)
}

func (rb *ResponseBodySync) IsFail() bool {
	return isFail(&rb.ResponseBodyBase)
}

func (rb *ResponseBodySync) IsError() bool {
	return isError(&rb.ResponseBodyBase)
}

func (rb *ResponseBodySync) SetMessage(msg string) {
	setMessage(&rb.ResponseBodyBase, msg)
}

func (rb *ResponseBodySync) GetMessage() string {
	return getMessage(&rb.ResponseBodyBase)
}

func (rb *ResponseBodySync) GetStatus() string {
	return getStatus(&rb.ResponseBodyBase)
}
