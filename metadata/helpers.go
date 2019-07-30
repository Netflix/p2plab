package metadata

import (
	"time"

	bolt "go.etcd.io/bbolt"
)

type bktTimestamp struct {
	key       []byte
	timestamp *time.Time
}

func ReadTimestamps(bkt *bolt.Bucket, created, updated *time.Time) error {
	for _, t := range []bktTimestamp{
		{bucketKeyCreatedAt, created},
		{bucketKeyUpdatedAt, updated},
	} {
		v := bkt.Get(t.key)
		if v == nil {
			continue
		}

		err := t.timestamp.UnmarshalBinary(v)
		if err != nil {
			return err
		}
	}

	return nil
}

func WriteTimestamps(bkt *bolt.Bucket, created, updated time.Time) error {
	createdAt, err := created.MarshalBinary()
	if err != nil {
		return err
	}

	updatedAt, err := updated.MarshalBinary()
	if err != nil {
		return err
	}

	for _, f := range []field{
		{bucketKeyCreatedAt, createdAt},
		{bucketKeyUpdatedAt, updatedAt},
	} {
		err = bkt.Put(f.key, f.value)
		if err != nil {
			return err
		}
	}

	return nil
}
