//go:build windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

func fallbackLockNamespace() (string, error) {
	sid, err := currentProcessUserSID()
	if err != nil {
		return "", err
	}
	value, err := sid.String()
	if err != nil {
		return "", err
	}
	return "sid:" + value, nil
}
