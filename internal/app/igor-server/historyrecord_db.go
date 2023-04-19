// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import "gorm.io/gorm"

func dbCreateHistoryRecordTx(hr *HistoryRecord) error {
	return performDbTx(func(tx *gorm.DB) error {
		return dbCreateHistoryRecord(hr, tx)
	})
}

func dbCreateHistoryRecord(hr *HistoryRecord, tx *gorm.DB) error {
	result := tx.Create(&hr)
	return result.Error
}
