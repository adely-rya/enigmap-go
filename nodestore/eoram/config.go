package eoram

const (
	defaultLevelsPerPack = 3
	defaultCacheCapacity = 1024
	defaultDataFile      = "/dev/shm/enigmap/oram-large-buckets.db"
)

type config struct {
	levelsPerPack uint64
	cacheCapacity int
	dataFile      string
}

func defaultConfig() config {
	return config{
		levelsPerPack: defaultLevelsPerPack,
		cacheCapacity: defaultCacheCapacity,
		dataFile:      defaultDataFile,
	}
}

// Option configures the ENIGMAP-specific Path ORAM node store.
type Option func(*config)

// WithLevelsPerPack sets the number of ORAM tree levels stored in one Large
// Bucket.
func WithLevelsPerPack(levels uint64) Option {
	return func(config *config) {
		config.levelsPerPack = levels
	}
}

// WithCacheCapacity sets the maximum number of decrypted Large Buckets kept
// in memory.
func WithCacheCapacity(capacity int) Option {
	return func(config *config) {
		config.cacheCapacity = capacity
	}
}

// WithDataFile sets the file used by the encrypted Large Bucket storage.
func WithDataFile(path string) Option {
	return func(config *config) {
		config.dataFile = path
	}
}
