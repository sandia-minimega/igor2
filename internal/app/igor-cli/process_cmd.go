// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"igor2/internal/pkg/api"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"igor2/internal/pkg/common"
)

const (
	CliUserAgentName = "IgorCLI"
)

var (
	lastAccessUser string
	termVal        = os.Getenv("TERM")
	_, envNoColor  = os.LookupEnv("NO_COLOR")
)

// doSend calls the appropriate method handler to send a request to igor-server
// and hands back the raw bytes of the HTTP response body.
func doSend(action string, apiPath string, params map[string]interface{}) *[]byte {

	endPoint := cli.IgorServerAddr + apiPath
	osUser, _ := user.Current()
	lastUser, _ := readLastAccessUser(osUser)
	if lastUser != "" {
		lastAccessUser = lastUser
	}

	var body *[]byte
	switch action {
	case http.MethodGet, http.MethodDelete:
		_, _, body = processRequestWithNoBody(action, endPoint)
	case http.MethodPost, http.MethodPatch, http.MethodPut:
		_, _, body = processRequestWithBody(action, endPoint, params)
	default:
		checkClientErr(fmt.Errorf("BAD ACTION RECEIVED - actions allowed: get, post, patch, put, delete"))
	}

	return body
}

func doSendMultiform(action string, apiPath string, params map[string]interface{}) *[]byte {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	endPoint := cli.IgorServerAddr + apiPath

	// fill the form with the map content
	for key, r := range params {
		var fw io.Writer
		var err error
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		// Add a file
		if x, ok := r.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				checkClientErr(err)
			}
			if _, err = io.Copy(fw, x); err != nil {
				checkClientErr(err)
			}
			// Add string fields
		} else if x, ok := r.(string); ok {
			w.WriteField(key, x)
			// Add string fields from []string
		} else if x, ok := r.([]string); ok {
			for _, s := range x {
				w.WriteField(key, s)
			}
		} else {
			checkClientErr(fmt.Errorf("unrecognized input type for %v: %T", key, r))
		}

	}

	// make sure to close the writer when done
	err := w.Close()
	if err != nil {
		checkClientErr(err)
	}

	// pack it into the request
	req, err := http.NewRequest(action, endPoint, &b)
	if err != nil {
		checkClientErr(err)
	}
	req.Header.Set(common.ContentType, w.FormDataContentType())

	_, _, body := doRequest(req)

	return body
}

func processRequestWithBody(method string, endPoint string, params map[string]interface{}) (string, http.Header, *[]byte) {
	reqData, err := json.Marshal(params)
	if err != nil {
		checkClientErr(err)
	}
	req, err := http.NewRequest(method, endPoint, bytes.NewBuffer(reqData))
	if err != nil {
		checkClientErr(err)
	}
	req.Header.Set(common.ContentType, common.MAppJson)
	return doRequest(req)
}

func processRequestWithNoBody(method string, endPoint string) (string, http.Header, *[]byte) {
	req, err := http.NewRequest(method, endPoint, nil)
	if err != nil {
		checkClientErr(err)
	}
	return doRequest(req)
}

func setAuthToken(r *http.Request) {
	// set the auth token, if not available simply don't include it
	if authToken, err := readAuthToken(); err == nil {
		r.Header.Set(common.Authorization, fmt.Sprintf("Bearer %v", authToken))
	}
}

func setUserAgent(r *http.Request) {
	version := common.GetVersion("", true)
	version = version[strings.Index(version, " ")+1:]
	version = version[0:strings.Index(version, " ")]
	r.Header.Set(common.UserAgent, CliUserAgentName+"/"+version)
}

func doRequest(req *http.Request) (string, http.Header, *[]byte) {

	setUserAgent(req)
	setAuthToken(req)
	resp := sendRequest(req)
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		checkClientErr(readErr)
	}
	return resp.Status, resp.Header, &body
}

func sendRequest(req *http.Request) *http.Response {
	client := getClient()
	resp, err := client.Do(req)
	if err != nil {
		if !connProblem(err) {
			checkClientErr(err)
		}
	}
	if err = writeLastAccessDate(); err != nil {
		fmt.Fprintf(os.Stderr, "problem writing to last access file : %v", err)
	}

	return resp
}

func getClient() *http.Client {

	var cert tls.Certificate
	var certErr error
	var caCertPool *x509.CertPool

	if cli.Client.CertFile != "" && cli.Client.KeyFile != "" {
		cert, certErr = tls.LoadX509KeyPair(cli.Client.CertFile, cli.Client.KeyFile)
		if certErr != nil {
			checkClientErr(fmt.Errorf("error creating x509 keypair from %s and %s", cli.Client.CertFile, cli.Client.KeyFile))
		}

		caCert, err := os.ReadFile(cli.Client.CaCert)
		if err != nil {
			fmt.Errorf("error reading CA cert file %s: %s", cli.Client.CaCert, err)
		}
		caCertPool = x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates:       []tls.Certificate{cert},
				RootCAs:            caCertPool,
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true,
			},
			TLSHandshakeTimeout: time.Second * 5,
			MaxIdleConns:        100,
			MaxConnsPerHost:     100,
			MaxIdleConnsPerHost: 100,
			Proxy:               http.ProxyFromEnvironment,
		},
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			setUserAgent(r)
			return clientRedirectHandler(r, via)
		},
		Timeout: time.Minute * 3,
	}

	return client
}

func writeLastAccessDate() error {
	osUser, _ := user.Current()
	lastAccessPath := filepath.Join(osUser.HomeDir, ".igor", "lastaccess")

	lastAccess := getLocTime(time.Now()).Format(time.RFC1123)
	f, err := os.OpenFile(lastAccessPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		return err
	}
	_, err = f.WriteString(lastAccess)
	if err != nil {
		return err
	}
	return nil
}

func writeLastAccessUser() error {
	osUser, _ := user.Current()
	lastUserPath := filepath.Join(osUser.HomeDir, ".igor", "lastuser")

	f, err := os.OpenFile(lastUserPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		return err
	}
	_, err = f.WriteString(lastAccessUser)
	if err != nil {
		return err
	}
	return nil
}

func getAuthTokenPath() string {
	osUser, _ := user.Current()
	return filepath.Join(osUser.HomeDir, ".igor", "auth_token")
}

func writeAuthToken(cookie *http.Cookie) error {
	authTokenPath := getAuthTokenPath()
	f, err := os.OpenFile(authTokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		return err
	}
	_, err = f.WriteString(cookie.Value)
	if err != nil {
		return err
	}
	return nil
}

func readAuthToken() (string, error) {
	authTokenPath := getAuthTokenPath()
	token, err := os.ReadFile(authTokenPath)
	if err != nil || len(token) == 0 {
		return "", fmt.Errorf("token not available - %v", err)
	}
	return string(token), nil
}

func readLastAccessUser(osUser *user.User) (string, error) {
	lastUserPath := filepath.Join(osUser.HomeDir, ".igor", "lastuser")
	lastUser, err := os.ReadFile(lastUserPath)
	if err != nil {
		return "", fmt.Errorf("last user information not available - have you logged in at least once?")
	}
	return string(lastUser), nil
}

func getLastAccessInfo() (string, error) {
	osUser, _ := user.Current()
	lastUser, err := readLastAccessUser(osUser)
	if err != nil {
		return "", err
	}

	lastAccessPath := filepath.Join(osUser.HomeDir, ".igor", "lastaccess")
	date, err := os.ReadFile(lastAccessPath)
	if err != nil {
		return "", fmt.Errorf("last access information not available - have you logged in at least once?")
	}

	return fmt.Sprintf("last access by %s as '%s' at %s.\nYour next command will be as '%s' "+
		"if your token hasn't expired. Use 'igor logout' to change this.",
		osUser.Name, lastUser, date, lastUser), nil
}

// clientRedirectHandler examines a request/response that was redirected and takes action
// to move to the next step. Mainly this is used to handle login cases when the
// user's auth token needs to be created or refreshed with user creds before continuing
// with the original action.
func clientRedirectHandler(req *http.Request, via []*http.Request) error {

	osUser, osErr := user.Current()
	if osErr != nil {
		return osErr
	}

	// if the redirect path is to /login, request the user's creds and change set the params
	// necessary to make the proper call
	if req.URL.Path == api.Login {
		username, password, err := reqUserCreds(osUser)
		if err != nil {
			return err
		}
		lastAccessUser = username
		req.Method = "POST"
		req.SetBasicAuth(username, password)
		return nil
	}

	// if the redirect preceding this one was to /login and the response contains a
	// cookie then save the auth token and use the original URL that should have been
	// returned by the redirect. also remove the referer header.
	if via[len(via)-1].URL.Path == api.Login {
		cookies := req.Response.Cookies()
		if len(cookies) == 0 {
			return fmt.Errorf("no cookie returned after login")
		} else {
			for _, c := range cookies {
				if c.Name == "auth_token" {
					err := writeAuthToken(cookies[0])
					if err != nil {
						return err
					}
					writeLastAccessUser()
				}
			}

			if req.URL.Path == via[0].URL.Path {
				req.Method = via[0].Method
				req.Header.Del(common.Referer)
				setAuthToken(req)
			}
		}
	}

	return nil
}

func unmarshalBasicResponse(body *[]byte) *common.ResponseBodyBasic {
	rb := &common.ResponseBodyBasic{}
	err := json.Unmarshal(*body, rb)
	checkUnmarshalErr(err)
	return rb
}

// checkUnmarshalErr prints a message if the unmarshaling the response body failed
func checkUnmarshalErr(err error) {
	if err != nil {
		newErr := fmt.Errorf("unable to interpret server response (notify admin) - %v", err)
		checkClientErr(newErr)
	}
}

// checkRespFailure returns true if the response is missing or indicates a problem
func checkRespFailure(rb common.ResponseBody) bool {
	if rb.IsFail() || rb.IsError() {
		return true
	}
	return false
}

func connProblem(err error) bool {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			checkClientErr(fmt.Errorf("connection timeout"))
		}
		var opErr *net.OpError
		if errors.As(urlErr.Err, &opErr) {
			var scErr *os.SyscallError
			if errors.As(opErr.Err, &scErr) {
				if scErr.Err == syscall.ECONNREFUSED {
					checkClientErr(fmt.Errorf("connection refused -- check igor-server address... also is igor-server running?"))
				} else {
					checkClientErr(scErr.Err)
				}
			}
		}
	}
	return false
}
