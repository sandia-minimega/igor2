// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorcli

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gookit/color"
	"github.com/spf13/pflag"

	"igor2/internal/pkg/common"
)

var testHostBasic = common.HostData{
	Name:         "",
	SequenceID:   0,
	Eth:          "",
	IP:           "",
	Mac:          "00:00:00:00:00:00",
	State:        "available",
	Powered:      "false",
	Cluster:      "",
	HostPolicy:   "default",
	AccessGroups: []string{"all"},
	Restricted:   false,
	Reservations: nil,
}

func generateTestHosts(count int, cluster *common.ClusterData) []common.HostData {
	testHosts := make([]common.HostData, count)
	for i := 0; i < count; i++ {
		testHosts[i] = testHostBasic
		testHosts[i].Cluster = cluster.Name
		testHosts[i].Name = cluster.Prefix + strconv.Itoa(i+1)
		testHosts[i].SequenceID = i + 1
	}
	return testHosts
}

func getSomeData() *common.ResponseBodyShow {

	lastAccessUser = "tombomb"

	c := common.ClusterData{
		Name:          "krypton",
		Prefix:        "kn",
		DisplayWidth:  16,
		DisplayHeight: 3,
		Motd:          "This is a test cluster",
		MotdUrgent:    true,
	}

	h := generateTestHosts(48, &c)

	// kn1
	h[0].State = strings.ToLower(Reserved)
	h[1].Powered = "true"
	h[1].State = strings.ToLower(Reserved)
	h[2].Powered = "true"
	h[2].State = strings.ToLower(Reserved)
	// kn4
	h[3].Powered = "unknown"
	// kn5
	h[4].HostPolicy = "weekdays-only"
	h[4].Restricted = true
	// kn6
	h[5].Powered = "false"
	h[6].State = strings.ToLower(Reserved)
	h[6].Powered = "true"
	// kn8
	h[7].State = strings.ToLower(Blocked)
	// kn9
	h[8].Powered = "false"
	h[8].State = strings.ToLower(Blocked)
	// kn11
	h[9].Powered = "unknown"
	h[9].State = strings.ToLower(Blocked)
	h[10].State = strings.ToLower(Reserved)
	h[10].Powered = "false"
	// kn11
	h[10].HostPolicy = "glados-only"
	h[10].AccessGroups = []string{"ApertureLabs"}

	//
	h[12].State = strings.ToLower(Reserved)
	h[12].Powered = "true"
	h[14].State = strings.ToLower(Reserved)
	h[14].Powered = "true"

	h[15].State = strings.ToLower(Reserved)
	h[15].Powered = "true"
	h[16].State = strings.ToLower(Reserved)
	h[16].Powered = "true"
	h[17].State = strings.ToLower(Reserved)
	h[17].Powered = "true"

	h[22].State = strings.ToLower(Reserved)
	h[22].Powered = "true"
	h[23].State = strings.ToLower(Reserved)
	h[23].Powered = "true"

	h[26].State = strings.ToLower(Reserved)
	h[27].State = strings.ToLower(Reserved)
	h[29].State = strings.ToLower(Reserved)
	//h[29].Powered = "unknown"
	// kn21
	// kn22

	r := make([]common.ReservationData, 0, 10)

	r = append(r, common.ReservationData{
		Name:         "willow",
		Description:  "",
		Owner:        "tombomb",
		Group:        "",
		Profile:      "",
		Distro:       "",
		Vlan:         0,
		Start:        time.Now().Add(-time.Minute * time.Duration(rand.Intn(7*60+1)+1)).Unix(),
		End:          time.Now().Add(time.Minute * time.Duration(rand.Intn(24*60-1)+1)).Unix(),
		OrigEnd:      0,
		ExtendCount:  0,
		Hosts:        []string{"kn1", "kn2", "kn3"},
		HostRange:    "kn[1-3]",
		HostsUp:      "kn[2-3]",
		HostsDown:    "kn1",
		Installed:    true,
		InstallError: "",
		RemainHours:  0,
	})
	r = append(r, common.ReservationData{
		Name:         "yellow-boots",
		Description:  "",
		Owner:        "tombomb",
		Group:        "",
		Profile:      "",
		Distro:       "",
		Vlan:         0,
		Start:        time.Now().Add(-time.Minute * time.Duration(rand.Intn(3*60+1)+1)).Unix(),
		End:          time.Now().Add(time.Minute * time.Duration(rand.Intn(24*60-1)+1)).Unix(),
		OrigEnd:      0,
		ExtendCount:  0,
		Hosts:        []string{"kn7"},
		HostsUp:      "kn7",
		HostRange:    "kn7",
		Installed:    true,
		InstallError: "",
		RemainHours:  0,
	})

	r = append(r, common.ReservationData{
		Name:         "test-chamber",
		Description:  "",
		Owner:        "glados",
		Group:        "",
		Profile:      "",
		Distro:       "",
		Vlan:         0,
		Start:        time.Now().Add(time.Minute * time.Duration(rand.Intn(7*60+1)+1)).Add(time.Hour * 24 * 378).Unix(),
		End:          time.Now().Add(time.Minute * time.Duration(rand.Intn(24*60-1)+1)).Add(time.Hour * 24 * 378).Unix(),
		OrigEnd:      0,
		ExtendCount:  0,
		Hosts:        []string{"kn5", "kn11"},
		HostRange:    "kn[5,11]",
		HostsUp:      "kn[5,11]",
		Installed:    false,
		InstallError: "",
		RemainHours:  0,
	})

	startTime := time.Now().Add(time.Hour * time.Duration(rand.Intn(24*7+1)+24))
	endTime := startTime.Add(time.Hour * time.Duration(rand.Intn(24*7+1)+1))

	r = append(r, common.ReservationData{
		Name:         "OldForest",
		Description:  "",
		Owner:        "tombomb",
		Group:        "",
		Profile:      "",
		Distro:       "",
		Vlan:         0,
		Start:        startTime.Unix(),
		End:          endTime.Unix(),
		OrigEnd:      0,
		ExtendCount:  0,
		Hosts:        []string{"kn12"},
		HostRange:    "kn12",
		HostsUp:      "kn12",
		Installed:    false,
		InstallError: "",
		RemainHours:  0,
	})

	r = append(r, common.ReservationData{
		Name:         "hunted",
		Description:  "",
		Owner:        "chell",
		Group:        "",
		Profile:      "",
		Distro:       "",
		Vlan:         0,
		Start:        time.Now().Add(-time.Minute * time.Duration(rand.Intn(7*60+1)+1)).Unix(),
		End:          time.Now().Add(time.Minute * time.Duration(rand.Intn(24*60-1)+1)).Unix(),
		OrigEnd:      0,
		ExtendCount:  0,
		Hosts:        []string{"kn13", "kn15"},
		HostRange:    "kn[13,15]",
		HostsUp:      "kn[13,15]",
		Installed:    true,
		InstallError: "",
		RemainHours:  0,
	})

	r = append(r, common.ReservationData{
		Name:         "mojo",
		Description:  "",
		Owner:        "minime",
		Group:        "TheGoodGuys",
		Profile:      "",
		Distro:       "",
		Vlan:         0,
		Start:        time.Now().Add(-time.Minute * time.Duration(rand.Intn(7*60+1)+1)).Unix(),
		End:          time.Now().Add(time.Minute * time.Duration(rand.Intn(24*60-1)+1)).Unix(),
		OrigEnd:      0,
		ExtendCount:  0,
		Hosts:        []string{"kn16", "kn17", "kn18"},
		HostRange:    "kn[16-18]",
		HostsUp:      "kn[16-18]",
		Installed:    true,
		InstallError: "",
		RemainHours:  0,
	})

	r = append(r, common.ReservationData{
		Name:         "GiantBomb",
		Description:  "",
		Owner:        "minime",
		Group:        "TheGoodGuys",
		Profile:      "",
		Distro:       "",
		Vlan:         0,
		Start:        time.Now().Add(-time.Minute * time.Duration(rand.Intn(7*60+1)+1)).Unix(),
		End:          time.Now().Add(time.Minute * time.Duration(rand.Intn(24*60-1)+1)).Unix(),
		OrigEnd:      0,
		ExtendCount:  0,
		Hosts:        []string{"kn23", "kn24"},
		HostRange:    "kn[23-24]",
		HostsUp:      "kn[23-24]",
		Installed:    false,
		InstallError: "error!",
		RemainHours:  0,
	})

	r = append(r, common.ReservationData{
		Name:         "barrow",
		Description:  "",
		Owner:        "tombomb",
		Group:        "",
		Profile:      "",
		Distro:       "",
		Vlan:         0,
		Start:        time.Now().Add(-time.Minute * time.Duration(rand.Intn(7*60+1)+1)).Unix(),
		End:          time.Now().Add(time.Minute * time.Duration(rand.Intn(24*60-1)+1)).Unix(),
		OrigEnd:      0,
		ExtendCount:  0,
		Hosts:        []string{"kn27", "kn28", "kn30"},
		HostRange:    "kn[27-28,30]",
		HostsUp:      "",
		HostsDown:    "kn[27-28,30]",
		HostsPowerNA: "",
		Installed:    true,
		InstallError: "",
		RemainHours:  0,
	})

	showData := common.ShowData{
		Cluster:      c,
		Hosts:        h,
		Reservations: r,
		UserGroups:   []string{"all", "TheGoodGuys"},
	}

	rb := &common.ResponseBodyShow{
		ResponseBodyBase: common.ResponseBodyBase{
			Status:     "",
			Message:    "",
			ServerTime: time.Now().Format(common.DateTimeServerFormat),
		},
		Data: map[string]common.ShowData{"show": showData},
	}

	return rb
}

func resetGlobalTestVars() {
	cli.tzLoc, _ = time.LoadLocation("America/Los_Angeles")
	igorCliNow = getLocTime(time.Now())
	simplePrint = false
	noColor = false
	color.ResetOptions()
}

func TestShowNormal(t *testing.T) {

	resetGlobalTestVars()
	fmt.Printf("\nTest: 'igor show'\n\n")

	rb := getSomeData()
	flagset := &pflag.FlagSet{}
	//flagset.Bool("all", false, "")
	var args []string
	if err := flagset.Parse(args); err != nil {
		t.Fatal(err)
	}
	printShow(rb, flagset)
}

func TestShowAll(t *testing.T) {

	resetGlobalTestVars()
	fmt.Printf("\nTest: 'igor show --all'\n\n")

	rb := getSomeData()
	flagset := &pflag.FlagSet{}
	flagset.Bool("all", true, "")
	args := []string{"--all"}
	if err := flagset.Parse(args); err != nil {
		t.Fatal(err)
	}
	printShow(rb, flagset)
}

func TestShowAllSimple(t *testing.T) {

	resetGlobalTestVars()
	fmt.Printf("\nTest: 'igor show --all --simple'\n\n")

	rb := getSomeData()
	flagset := &pflag.FlagSet{}
	flagset.Bool("simple", true, "")
	flagset.Bool("all", true, "")
	args := []string{"--all", "--simple"}
	if err := flagset.Parse(args); err != nil {
		t.Fatal(err)
	}
	printShow(rb, flagset)
}

func TestShowAllNoMapRemainTime(t *testing.T) {

	resetGlobalTestVars()
	fmt.Printf("\nTest: 'igor show -r --sort-start --all --no-map'\n\n")

	rb := getSomeData()
	flagset := &pflag.FlagSet{}
	flagset.Bool("no-map", true, "")
	flagset.Bool("all", true, "")
	flagset.Bool("remaining", true, "")
	flagset.Bool("sort-start", true, "")

	args := []string{"--sort-start", "--remaining", "--all", "--no-map"}
	if err := flagset.Parse(args); err != nil {
		t.Fatal(err)
	}

	printShow(rb, flagset)

}
