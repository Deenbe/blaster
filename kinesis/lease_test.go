/*
Copyright © 2020 Blaster Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kinesis

import "time"

type Lease struct {
	GroupID          string
	ResourceID       string
	OwnerID          string
	Term             uint64
	PreviousTerm     uint64
	LastHeartBeat    time.Time
	CheckpointOffset uint64
	CheckpointState  string
}

type OwnedLease struct {
	lease *Lease
	done  chan<- struct{}
}

func (ol *OwnedLease) Done() chan<- struct{} {
	return ol.done
}

type StateReaderWriter interface {
	Write(lease *Lease) (bool, error)
	Read(groupID, resourceID string) (*Lease, error)
	ReadGroup(groupID string) ([]*Lease, error)
}

type DynamoDBStateReaderWriter struct {
}

func (rw *DynamoDBStateReaderWriter) Write(lease *Lease) (bool, error) {
	panic("not implemented")
}

func (rw *DynamoDBStateReaderWriter) Read(groupID, resourceID string) (*Lease, error) {
	panic("not implemented")
}

func (rw *DynamoDBStateReaderWriter) ReadGroup(groupID string) ([]*Lease, error) {
	panic("not implemented")
}

type LeaseController interface {
	Acquire(groupID, resourceID, ownerID string) (chan struct{}, error)
}

type DefaultLeaseController struct {
	readerWriter StateReaderWriter
	now          func() time.Time
	heartbeat    chan<- time.Time
	leaseTimeout time.Duration
}

func (m *DefaultLeaseController) Acquire(groupID, resourceID, ownerID string) (*OwnedLease, error) {
	lease, err := m.readerWriter.Read(groupID, resourceID)
	if err != nil {
		return nil, err
	}
	if lease == nil {
		lease = &Lease{
			GroupID:    groupID,
			ResourceID: resourceID,
			OwnerID:    ownerID,
		}
	}

	// If the current lease is active we don't steal it.
	if m.now().Sub(lease.LastHeartBeat) < m.leaseTimeout {
		return nil, nil
	}

	lease.PreviousTerm = lease.Term
	lease.Term = lease.Term + 1
	lease.LastHeartBeat = m.now()
	success, err := m.readerWriter.Write(lease)
	if err != nil || !success {
		return nil, err
	}

	c := make(chan struct{})
	ol := &OwnedLease{
		lease: lease,
		done:  c,
	}

	m.startHeartbeat(ol)
	return ol, nil
}

func (m *DefaultLeaseController) startHeartbeat(ownedLease *OwnedLease) {
}

func NewLeaseController(stateReaderWriter StateReaderWriter) *DefaultLeaseController {
	return &DefaultLeaseController{
		readerWriter: stateReaderWriter,
	}
}
