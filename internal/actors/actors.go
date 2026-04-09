// SPDX-License-Identifier: BUSL-1.1

package actors

type Actor interface {
	Act(msg string) error
}
