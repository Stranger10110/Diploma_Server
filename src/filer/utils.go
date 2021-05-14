package filer

import "fmt"

func CheckSetCheckFileLock(fullRelPath string, errorPath string, firstCheckUuid bool) error {
	// Check file lock
	if _, lock := GetFileLock(fullRelPath); lock != "" && firstCheckUuid && lock != Uuid {
		return fmt.Errorf("%s is busy", errorPath)
	} else {
		// If no lock, set it and continue
		SetFileLock(fullRelPath)
		if _, lock2 := GetFileLock(fullRelPath); lock2 != Uuid {
			return fmt.Errorf("%s is busy", errorPath)
		} else {
			return nil
		}
	}
}
