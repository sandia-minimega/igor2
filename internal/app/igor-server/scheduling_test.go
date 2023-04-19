// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func getMaxEnd() time.Time {
	testNow := time.Date(2021, time.April, 1, 10, 0, 0, 0, time.Local)
	f := testNow.Add(time.Minute * time.Duration(1457*60*24))
	return time.Date(f.Year(), f.Month(), f.Day(), f.Hour(), f.Minute(), 0, 0, time.Local)
}

func TestNoContiguousBlocks(t *testing.T) {

	testNow := time.Date(2021, time.April, 1, 10, 0, 0, 0, time.Local)
	res1Start := time.Date(2021, time.April, 1, 2, 0, 0, 0, time.Local)
	res1Dur, _ := time.ParseDuration("6h30m")
	res2Start := time.Date(2021, time.March, 29, 16, 0, 0, 0, time.Local)
	res2End := time.Date(2021, time.April, 1, 9, 0, 0, 0, time.Local)
	res2Next := time.Date(2021, time.April, 2, 12, 0, 0, 0, time.Local)

	testSlots := []ReservationTimeSlot{
		{"kn9", 9, "", time.Time{}, testNow, "", getMaxEnd()},
		{"kn3", 3, "", time.Time{}, testNow, "", getMaxEnd()},
		{"kn7", 7, "res1", res1Start, res1Start.Add(res1Dur), "", getMaxEnd()},
		{"kn12", 12, "res1", res1Start, res1Start.Add(res1Dur), "", getMaxEnd()},
		{"kn14", 14, "res2", res2Start, res2End, "res3", res2Next},
	}
	testSlotsMap := map[string][]ReservationTimeSlot{}
	testSlotsMap[DefaultPolicyName] = testSlots

	hostNameList := findBestSolution(testSlotsMap, false, 5)

	assert.Contains(t, hostNameList, "kn3", "doesn't contain all nodes")
	assert.Contains(t, hostNameList, "kn9", "doesn't contain all nodes")
	assert.Contains(t, hostNameList, "kn7", "doesn't contain all nodes")
	assert.Contains(t, hostNameList, "kn14", "doesn't contain all nodes")
	assert.Contains(t, hostNameList, "kn12", "doesn't contain all nodes")

}

func TestChooseSmallerContiguousBlock(t *testing.T) {

	testNow := time.Date(2021, time.April, 1, 10, 0, 0, 0, time.Local)
	res1Start := time.Date(2021, time.April, 1, 2, 0, 0, 0, time.Local)
	res1Dur, _ := time.ParseDuration("6h30m")
	res2Start := time.Date(2021, time.March, 29, 16, 0, 0, 0, time.Local)
	res2End := time.Date(2021, time.April, 1, 9, 0, 0, 0, time.Local)
	res2Next := time.Date(2021, time.April, 2, 12, 0, 0, 0, time.Local)

	testSlots := []ReservationTimeSlot{
		{"kn8", 8, "", time.Time{}, testNow, "", getMaxEnd()},
		{"kn9", 9, "", time.Time{}, testNow, "", getMaxEnd()},
		{"kn22", 22, "", time.Time{}, testNow, "", getMaxEnd()},
		{"kn13", 13, "res1", res1Start, res1Start.Add(res1Dur), "", getMaxEnd()},
		{"kn12", 12, "res1", res1Start, res1Start.Add(res1Dur), "", getMaxEnd()},
		{"kn14", 14, "res2", res2Start, res2End, "res3", res2Next},
	}

	testSlotsMap := map[string][]ReservationTimeSlot{}
	testSlotsMap[DefaultPolicyName] = testSlots

	hostNameList := findBestSolution(testSlotsMap, false, 4)

	assert.Contains(t, hostNameList, "kn22", "doesn't contain all correct nodes")
	assert.Contains(t, hostNameList, "kn14", "doesn't contain all correct nodes")
	assert.Contains(t, hostNameList, "kn12", "doesn't contain all correct nodes")
	assert.Contains(t, hostNameList, "kn13", "doesn't contain all correct nodes")
	assert.NotContains(t, hostNameList, "kn8", "node should not be present")
	assert.NotContains(t, hostNameList, "kn9", "node should not be present")

	hostNameList = findBestSolution(testSlotsMap, false, 2)

	assert.NotContains(t, hostNameList, "kn22", "node should not be present")
	assert.NotContains(t, hostNameList, "kn14", "node should not be present")
	assert.NotContains(t, hostNameList, "kn12", "node should not be present")
	assert.NotContains(t, hostNameList, "kn13", "node should not be present")
	assert.Contains(t, hostNameList, "kn8", "doesn't contain all correct nodes")
	assert.Contains(t, hostNameList, "kn9", "doesn't contain all correct nodes")

}
