package metadata

import (
	"sort"
	"time"

	"github.com/Netflix/p2plab/errdefs"
	"github.com/pkg/errors"
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

func readMap(bkt *bolt.Bucket, name []byte) (map[string]string, error) {
	mbkt := bkt.Bucket(name)
	if mbkt == nil {
		return nil, nil
	}

	m := make(map[string]string)
	err := mbkt.ForEach(func(k, v []byte) error {
		m[string(k)] = string(v)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return m, nil
}

func writeMap(bkt *bolt.Bucket, name []byte, m map[string]string) error {
	// Remove existing map to prevent merging.
	mbkt := bkt.Bucket(name)
	if mbkt != nil {
		err := bkt.DeleteBucket(name)
		if err != nil {
			return err
		}
	}

	if len(m) == 0 {
		return nil
	}

	var err error
	mbkt, err = bkt.CreateBucket(name)
	if err != nil {
		return err
	}

	for k, v := range m {
		if v == "" {
			delete(m, k)
			continue
		}

		err := mbkt.Put([]byte(k), []byte(v))
		if err != nil {
			return errors.Wrapf(err, "failed to set key value %q=%q", k, v)
		}
	}

	return nil
}

type labelCallback func(bkt *bolt.Bucket, id string, labels []string) error

func writeLabels(bkt *bolt.Bucket, ids, addLabels, removeLabels []string, cb labelCallback) error {
	if len(ids) == 0 {
		return nil
	}

	addSet := make(map[string]struct{})
	for _, l := range addLabels {
		addSet[l] = struct{}{}
	}

	removeSet := make(map[string]struct{})
	for _, l := range removeLabels {
		removeSet[l] = struct{}{}
	}

	for _, id := range ids {
		ibkt := bkt.Bucket([]byte(id))
		if ibkt == nil {
			return errors.Wrapf(errdefs.ErrNotFound, "%q", id)
		}

		lbkt := ibkt.Bucket(bucketKeyLabels)

		var labels []string
		if lbkt != nil {
			err := lbkt.ForEach(func(k, v []byte) error {
				if _, ok := removeSet[string(k)]; ok {
					return nil
				}

				labels = append(labels, string(k))
				delete(addSet, string(k))
				return nil
			})
			if err != nil {
				return err
			}

			err = ibkt.DeleteBucket(bucketKeyLabels)
			if err != nil {
				return err
			}
		}

		for l, _ := range addSet {
			labels = append(labels, l)
		}
		sort.Strings(labels)

		var err error
		lbkt, err = ibkt.CreateBucket(bucketKeyLabels)
		if err != nil {
			return err
		}

		err = cb(ibkt, id, labels)
		if err != nil {
			return err
		}
	}

	return nil
}
