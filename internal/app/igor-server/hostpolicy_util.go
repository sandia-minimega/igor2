// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

// hostPolicyIDsOfHostPolicies returns a list of HostPolicy IDs from
// the provided list of host policies.
func hostPolicyIDsOfHostPolicies(policies []HostPolicy) []int {
	hpIDs := make([]int, len(policies))
	for i, hp := range policies {
		hpIDs[i] = hp.ID
	}
	return hpIDs
}

// hostPolicyNamesOfHostPolicies returns a list of HostPolicy names from
// the provided list of host policies.
func hostPolicyNamesOfHostPolicies(policies []HostPolicy) []string {
	hpNames := make([]string, len(policies))
	for i, hp := range policies {
		hpNames[i] = hp.Name
	}
	return hpNames
}

// checkHostpolicyNameRules validates the host policy name
func checkHostpolicyNameRules(ref string) error {
	err := checkGenericNameRules(ref)
	if err != nil {
		return err
	}
	return nil
}

func getHostPoliciesFromHostNames(hostNames []string) ([]HostPolicy, error) {
	hpParams := map[string]interface{}{}
	if hostIDs, _, err := getHostIDsFromNames(hostNames); err != nil {
		return nil, err
	} else {
		hpParams["hosts"] = hostIDs
	}
	myHostPolicies, err := dbReadHostPoliciesTx(hpParams, &logger)
	if err != nil {
		return nil, err
	}
	return myHostPolicies, nil
}
