// Licensed to the LF AI & Data foundation under one
// or more contributor license agreements. See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership. The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metacache

import (
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/samber/lo"

	"github.com/milvus-io/milvus/internal/storage"
	"github.com/milvus-io/milvus/pkg/util/paramtable"
)

// BloomFilterSet is a struct with multiple `storage.PkStatstics`.
// it maintains bloom filter generated from segment primary keys.
// it may be updated with new insert FieldData when serving growing segments.
type BloomFilterSet struct {
	mut       sync.Mutex
	batchSize uint
	current   *storage.PkStatistics
	history   []*storage.PkStatistics
}

// NewBloomFilterSet returns a BloomFilterSet with provided historyEntries.
// Shall serve Flushed segments only. For growing segments, use `NewBloomFilterSetWithBatchSize` instead.
func NewBloomFilterSet(historyEntries ...*storage.PkStatistics) *BloomFilterSet {
	return &BloomFilterSet{
		batchSize: paramtable.Get().CommonCfg.BloomFilterSize.GetAsUint(),
		history:   historyEntries,
	}
}

// NewBloomFilterSetWithBatchSize returns a BloomFilterSet.
// The batchSize parameter is used to initialize new bloom filter.
// It shall be the estimated row count per batch for segment to sync with.
func NewBloomFilterSetWithBatchSize(batchSize uint, historyEntries ...*storage.PkStatistics) *BloomFilterSet {
	return &BloomFilterSet{
		batchSize: batchSize,
		history:   historyEntries,
	}
}

func (bfs *BloomFilterSet) PkExists(pk storage.PrimaryKey) bool {
	bfs.mut.Lock()
	defer bfs.mut.Unlock()
	if bfs.current != nil && bfs.current.PkExist(pk) {
		return true
	}

	for _, bf := range bfs.history {
		if bf.PkExist(pk) {
			return true
		}
	}
	return false
}

func (bfs *BloomFilterSet) UpdatePKRange(ids storage.FieldData) error {
	bfs.mut.Lock()
	defer bfs.mut.Unlock()

	if bfs.current == nil {
		bfs.current = &storage.PkStatistics{
			PkFilter: bloom.NewWithEstimates(bfs.batchSize,
				paramtable.Get().CommonCfg.MaxBloomFalsePositive.GetAsFloat()),
		}
	}

	return bfs.current.UpdatePKRange(ids)
}

func (bfs *BloomFilterSet) Roll(newStats ...*storage.PrimaryKeyStats) {
	bfs.mut.Lock()
	defer bfs.mut.Unlock()

	if len(newStats) > 0 {
		bfs.history = append(bfs.history, lo.Map(newStats, func(stats *storage.PrimaryKeyStats, _ int) *storage.PkStatistics {
			return &storage.PkStatistics{
				PkFilter: stats.BF,
				MaxPK:    stats.MaxPk,
				MinPK:    stats.MinPk,
			}
		})...)
		bfs.current = nil
	}
}

func (bfs *BloomFilterSet) GetHistory() []*storage.PkStatistics {
	bfs.mut.Lock()
	defer bfs.mut.Unlock()

	return bfs.history
}
