/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package id

// ReplicaIDProvider supplies the stable replica identifier used as the locker owner.
type ReplicaIDProvider interface {
	ID() string
}
