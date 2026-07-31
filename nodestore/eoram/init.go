package eoram

// initializeTree fills every physical ORAM bucket with fixed-size dummy blocks.
// Initialization is intentionally non-oblivious because it is outside the
// measured experiment.
func (client *Client[T]) initializeTree() error {
	var dummyData NodeBlock[T]
	emptyBucket := make(Bucket[NodeBlock[T]], client.bucketSize)
	for slot := range emptyBucket {
		emptyBucket[slot] = Block[NodeBlock[T]]{
			Data:  dummyData,
			Valid: 0,
		}
	}

	type initializer interface {
		Initialize(Bucket[NodeBlock[T]]) error
	}
	if storage, ok := client.server.Storage.(initializer); ok {
		return storage.Initialize(emptyBucket)
	}

	for bucketIndex := uint64(0); bucketIndex < client.server.Storage.Size(); bucketIndex++ {
		if err := client.server.Storage.Write(emptyBucket, bucketIndex); err != nil {
			return err
		}
	}

	return nil
}
