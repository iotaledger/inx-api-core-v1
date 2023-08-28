package engine

import (
	"runtime"

	"github.com/iotaledger/hive.go/kvstore/rocksdb"
)

// NewRocksDB creates a new RocksDB instance.
func NewRocksDB(path string, readonly bool) (*rocksdb.RocksDB, error) {
	if readonly {
		return rocksdb.OpenDBReadOnly(path,
			rocksdb.Custom([]string{
				"max_open_files=-1", // set max_open_files to -1 to always keep all files open, which avoids expensive table cache calls.
			}))
	}

	opts := []rocksdb.Option{
		rocksdb.IncreaseParallelism(runtime.NumCPU() - 1),
		rocksdb.Custom([]string{
			"periodic_compaction_seconds=43200",
			"level_compaction_dynamic_level_bytes=true",
			"keep_log_file_num=2",
			"max_log_file_size=50000000", // 50MB per log file
		}),
	}

	return rocksdb.CreateDB(path, opts...)
}
