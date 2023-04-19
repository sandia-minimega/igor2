// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

// IResInstaller is an interface that provides the mechanism for installing and uninstalling reservation OS images on cluster nodes.
type IResInstaller interface {
	// Install activates a reservation
	Install(*Reservation) error

	// Uninstall deactivates a reservation
	Uninstall(*Reservation) error
}
