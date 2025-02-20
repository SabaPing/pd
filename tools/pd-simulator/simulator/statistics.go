// Copyright 2018 TiKV Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package simulator

import (
	"fmt"
	"math"

	"github.com/tikv/pd/pkg/syncutil"
)

type taskStatistics struct {
	syncutil.RWMutex
	addVoter       map[uint64]int
	removePeer     map[uint64]int
	addLearner     map[uint64]int
	promoteLeaner  map[uint64]int
	demoteVoter    map[uint64]int
	transferLeader map[uint64]map[uint64]int
	mergeRegion    int
}

func newTaskStatistics() *taskStatistics {
	return &taskStatistics{
		addVoter:       make(map[uint64]int),
		removePeer:     make(map[uint64]int),
		addLearner:     make(map[uint64]int),
		promoteLeaner:  make(map[uint64]int),
		demoteVoter:    make(map[uint64]int),
		transferLeader: make(map[uint64]map[uint64]int),
	}
}

func (t *taskStatistics) getStatistics() map[string]int {
	t.RLock()
	defer t.RUnlock()
	stats := make(map[string]int)
	addVoter := getSum(t.addVoter)
	removePeer := getSum(t.removePeer)
	addLearner := getSum(t.addLearner)
	promoteLearner := getSum(t.promoteLeaner)
	demoteVoter := getSum(t.demoteVoter)

	var transferLeader int
	for _, to := range t.transferLeader {
		for _, v := range to {
			transferLeader += v
		}
	}

	stats["Add Voter (task)"] = addVoter
	stats["Remove Peer (task)"] = removePeer
	stats["Add Learner (task)"] = addLearner
	stats["Promote Learner (task)"] = promoteLearner
	stats["Demote Voter (task)"] = demoteVoter
	stats["Transfer Leader (task)"] = transferLeader
	stats["Merge Region (task)"] = t.mergeRegion

	return stats
}

func (t *taskStatistics) incAddVoter(regionID uint64) {
	t.Lock()
	defer t.Unlock()
	t.addVoter[regionID]++
}

func (t *taskStatistics) incAddLearner(regionID uint64) {
	t.Lock()
	defer t.Unlock()
	t.addLearner[regionID]++
}

func (t *taskStatistics) incPromoteLearner(regionID uint64) {
	t.Lock()
	defer t.Unlock()
	t.promoteLeaner[regionID]++
}

func (t *taskStatistics) incDemoteVoter(regionID uint64) {
	t.Lock()
	defer t.Unlock()
	t.demoteVoter[regionID]++
}

func (t *taskStatistics) incRemovePeer(regionID uint64) {
	t.Lock()
	defer t.Unlock()
	t.removePeer[regionID]++
}

func (t *taskStatistics) incMergeRegion() {
	t.Lock()
	defer t.Unlock()
	t.mergeRegion++
}

func (t *taskStatistics) incTransferLeader(fromPeerStoreID, toPeerStoreID uint64) {
	t.Lock()
	defer t.Unlock()
	_, ok := t.transferLeader[fromPeerStoreID]
	if ok {
		t.transferLeader[fromPeerStoreID][toPeerStoreID]++
	} else {
		m := make(map[uint64]int)
		m[toPeerStoreID]++
		t.transferLeader[fromPeerStoreID] = m
	}
}

type snapshotStatistics struct {
	syncutil.RWMutex
	receive map[uint64]int
	send    map[uint64]int
}

func newSnapshotStatistics() *snapshotStatistics {
	return &snapshotStatistics{
		receive: make(map[uint64]int),
		send:    make(map[uint64]int),
	}
}

type schedulerStatistics struct {
	taskStats     *taskStatistics
	snapshotStats *snapshotStatistics
}

func newSchedulerStatistics() *schedulerStatistics {
	return &schedulerStatistics{
		taskStats:     newTaskStatistics(),
		snapshotStats: newSnapshotStatistics(),
	}
}

func (s *snapshotStatistics) getStatistics() map[string]int {
	s.RLock()
	defer s.RUnlock()
	maxSend := getMax(s.send)
	maxReceive := getMax(s.receive)
	minSend := getMin(s.send)
	minReceive := getMin(s.receive)

	stats := make(map[string]int)
	stats["Send Maximum (snapshot)"] = maxSend
	stats["Receive Maximum (snapshot)"] = maxReceive
	if minSend != math.MaxInt32 {
		stats["Send Minimum (snapshot)"] = minSend
	}
	if minReceive != math.MaxInt32 {
		stats["Receive Minimum (snapshot)"] = minReceive
	}

	return stats
}

func (s *snapshotStatistics) incSendSnapshot(storeID uint64) {
	s.Lock()
	defer s.Unlock()
	s.send[storeID]++
}

func (s *snapshotStatistics) incReceiveSnapshot(storeID uint64) {
	s.Lock()
	defer s.Unlock()
	s.receive[storeID]++
}

// PrintStatistics prints the statistics of the scheduler.
func (s *schedulerStatistics) PrintStatistics() {
	task := s.taskStats.getStatistics()
	snap := s.snapshotStats.getStatistics()
	for t, count := range task {
		fmt.Println(t, count)
	}
	for s, count := range snap {
		fmt.Println(s, count)
	}
}

func getMax(m map[uint64]int) int {
	var max int
	for _, v := range m {
		if v > max {
			max = v
		}
	}
	return max
}

func getMin(m map[uint64]int) int {
	min := math.MaxInt32
	for _, v := range m {
		if v < min {
			min = v
		}
	}
	return min
}

func getSum(m map[uint64]int) int {
	var sum int
	for _, v := range m {
		sum += v
	}
	return sum
}
