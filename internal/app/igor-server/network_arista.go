// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	"igor2/internal/pkg/common"
)

func init() {
	if networkSetFuncs == nil {
		networkSetFuncs = make(map[string]func([]Host, int) error)
		networkClearFuncs = make(map[string]func([]Host) error)
		networkVlanFuncs = make(map[string]func() (map[string]string, error))
	}
	networkSetFuncs["arista"] = aristaSet
	networkClearFuncs["arista"] = aristaClear
	networkVlanFuncs["arista"] = aristaVlan
}

var aristaClearTemplate = `enable
configure terminal
interface {{ $.Eth }}
no switchport access vlan
switchport mode access`

var aristaSetTemplate = `enable
configure terminal
interface {{ $.Eth }}
switchport mode dot1q-tunnel
switchport access vlan {{ $.VLAN }}`

type AristaConfig struct {
	Eth  string
	VLAN int
}

// Issue the given commands via the specified URL, username, and password.
func aristaJSONRPC(user, password, URL string, commands []string) (map[string]interface{}, error) {
	logger.Debug().Msgf("url for arista: %v", URL)
	data, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "runCmds",
		"id":      1,
		"params":  map[string]interface{}{"version": 1, "cmds": commands},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal: %v", err)
	}

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		TLSHandshakeTimeout: time.Second * 5,
		MaxIdleConns:        100,
		MaxConnsPerHost:     100,
		MaxIdleConnsPerHost: 100,
	}

	client := &http.Client{
		Transport: t,
		//Timeout: time.Second * 30,
	}

	path := fmt.Sprintf("http://%s:%s@%s", user, password, URL)
	req, err := http.NewRequest("POST", path, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	req.Header.Set(common.ContentType, common.MAppJson)
	resp, err := client.Do(req)
	// resp, err := http.Post(path, "application/json", strings.NewReader(string(data)))
	if err != nil {
		// replace the password with a placeholder so that it doesn't show up
		// in error logs
		msg := strings.Replace(err.Error(), password, "<PASSWORD>", -1)
		return nil, fmt.Errorf("post failed: %v", msg)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("readall: %v", err)
	}
	result := make(map[string]interface{})
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling arista response body to json: %v - body received: %v", err, string(body))
	}

	return result, nil
}

func aristaSet(hosts []Host, vlan int) error {
	t := template.Must(template.New("set").Parse(aristaSetTemplate))

	for _, h := range hosts {
		var b bytes.Buffer
		c := &AristaConfig{
			Eth:  h.Eth,
			VLAN: vlan,
		}
		err := t.Execute(&b, c)
		if err != nil {
			return err
		}
		// now split b into strings with newlines
		commands := strings.Split(b.String(), "\n")
		logger.Debug().Msgf("aristaSet commands being sent: %v", commands)

		result, err := aristaJSONRPC(igor.Vlan.NetworkUser, igor.Vlan.NetworkPassword, igor.Vlan.NetworkURL, commands)
		if err != nil {
			return err
		}
		logger.Debug().Msgf("aristaSet response received: %v", result)
	}

	return nil
}

func aristaClear(hosts []Host) error {
	t := template.Must(template.New("set").Parse(aristaClearTemplate))

	for _, h := range hosts {
		var b bytes.Buffer
		c := &AristaConfig{
			Eth: h.Eth,
		}
		err := t.Execute(&b, c)
		if err != nil {
			return err
		}
		// now split b into strings with newlines
		commands := strings.Split(b.String(), "\n")
		logger.Debug().Msgf("aristaClear commands being sent: %v", commands)

		result, err := aristaJSONRPC(igor.Vlan.NetworkUser, igor.Vlan.NetworkPassword, igor.Vlan.NetworkURL, commands)
		if err != nil {
			return err
		}
		logger.Debug().Msgf("aristaClear response received: %v", result)
	}

	return nil
}

func aristaVlan() (map[string]string, error) {
	// get vlan mappings for the range we care about
	commands := []string{fmt.Sprintf("show vlan %v-%v", igor.Vlan.RangeMin, igor.Vlan.RangeMax)}
	res, err := aristaJSONRPC(igor.Vlan.NetworkUser, igor.Vlan.NetworkPassword, igor.Vlan.NetworkURL, commands)
	result := make(map[string]string)
	if err != nil {
		logger.Error().Msgf("error sending command to vlan service: %v", err.Error())
		return nil, err
	}
	// parse out the block of data we actually want from the response
	res2 := res["result"].([]interface{})
	res3 := res2[0].(map[string]interface{})
	data := res3["vlans"].(map[string]interface{})
	ethMap := make(map[string]string)
	for key, value := range data {
		logger.Debug().Msgf("arista key: %v", key)
		inter := value.(map[string]interface{})["interfaces"].(map[string]interface{})
		logger.Debug().Msgf("arista interface: %v", inter)
		for k := range inter {
			eth := strings.ReplaceAll(k, "Ethernet", "Et")
			ethMap[eth] = key
		}
	}
	keys := make([]string, len(ethMap))
	i := 0
	for k := range ethMap {
		keys[i] = k
		i++
	}
	hosts, err := dbReadHostsTx(map[string]interface{}{"eth": keys})
	if err != nil {
		return nil, err
	}
	for _, h := range hosts {
		result[h.Name] = ethMap[h.Eth]
	}

	return result, nil
}
