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
	GroupID       string
	ResourceID    string
	OwnerID       string
	Term          uint64
	LastHeartBeat time.Time
	State         string
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
}

func (m *DefaultLeaseController) Acquire(groupID, resourceID, ownerID string) (chan struct{}, error) {
	panic("not implemented")
}
