package database


import (
	"encoding/json"
	"fmt"
	"github.com/nalej/derrors"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
	"time"
)

// timeout to prevent an indefinite wait
const BboltTimeOut = 3

type BboltDB struct {
	Path string
	DB *bolt.DB
}

func (b * BboltDB) OpenRead(path string) derrors.Error {
	return b.openDB(path, true)
}

func (b * BboltDB) OpenWrite(path string) derrors.Error {
	return b.openDB(path, false)
}

// TODO: think about if it is better return *bolt.DB. It could exist a case where we need to reuse this
// If we decide on it, add this parameter if unsafe operations
func (b * BboltDB) openDB(path string, readOnly bool)  derrors.Error {
	db, err := bolt.Open(path, 0600, &bolt.Options{ReadOnly: readOnly, Timeout: BboltTimeOut * time.Second})
	if err != nil {
		log.Error().Str("Path", path).Msg("error opening bbolt database")
		return derrors.NewInternalError("error opening bbolt database")
	}
	b.DB = db

	return nil
}

func (b * BboltDB) Close() {
	if b.DB != nil {
		b.DB.Close()
	}
	b.DB = nil
}

// UnsafeGenericExist check if an element exists
func (b * BboltDB) UnsafeGenericExist(bucketName string, key []byte) (bool, error ) {

	var exists bool
	err := b.DB.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))
		if bk == nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", bucketName))
		}

		res := bk.Get(key)

		if res == nil {
			exists = false
		}else{
			exists = true
		}
		return nil
	})

	return exists, err
}

// UnsafeAdd adds a new element after check it not exists
func (b * BboltDB) UnsafeAdd(bucketName string, key []byte, toAdd interface{}) error {

	toAddBytes, err := json.Marshal(toAdd)
	if err != nil {
		return err
	}

	err =  b.DB.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to create bucket: %v", err))
		}

		// check if already exists
		res := bk.Get(key)

		exists := res != nil
		if exists {
			return derrors.NewAlreadyExistsError("element already exists")
		}

		if err := bk.Put(key, toAddBytes); err != nil {
			return derrors.NewInternalError("Cannot add new element")
		}
		return nil
	})

	return err
}

// UnsafeGet retrieves an element
func (b * BboltDB) UnsafeGet (bucketName string, key []byte, result * interface{}) error {

	err := b.DB.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(bucketName))
		if bk == nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", bucketName))
		}

		res := bk.Get(key)

		if res == nil {
			return derrors.NewNotFoundError("cannot get the element")
		}

		if err := json.Unmarshal(res, result); err != nil {
			return derrors.NewInternalError("error creating object")
		}

		return nil
	})

	return err
}