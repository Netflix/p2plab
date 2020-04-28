package metadata

import (
	"context"
	"os"
	"testing"

	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

func TestDB(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var testDir = "dbmetatest"
	db, cleanup := newTestDB(t, testDir)
	defer func() {
		if err := cleanup(); err != nil {
			t.Fatal(err)
		}
	}()
	var (
		testBucket = "test"
		testKey    = "testkey"
		testValue  = "testvalue"
	)
	// this should return an error as we have not populated the datastore
	if err := db.View(ctx, func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(testBucket))
		if bkt == nil {
			return nil
		}
		return errors.New("found bucket")
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Update(ctx, func(tx *bolt.Tx) error {
		bkt, err := tx.CreateBucket([]byte(testBucket))
		if err != nil {
			return err
		}
		return bkt.Put([]byte(testKey), []byte(testValue))
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.View(ctx, func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(testBucket))
		if bkt == nil {
			return errors.New("should have found bucket")
		}
		data := bkt.Get([]byte(testKey))
		if data == nil || string(data) != testValue {
			return errors.New("bad value ofund")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func newTestDB(t *testing.T, path string) (DB, func() error) {
	db, err := NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() error {
		if err := db.Close(); err != nil {
			return err
		}
		return os.RemoveAll(path)
	}
	return db, cleanup
}
