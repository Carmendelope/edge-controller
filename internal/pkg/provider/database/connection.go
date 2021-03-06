/*
 * Copyright 2019 Nalej
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
	// ReadOnly flag to indicate the option of the last Open. Flag needed to re-open y checkConnection
	ReadOnly bool
}

func (b * BboltDB) OpenRead() derrors.Error {
	return b.openDB(true)
}

func (b * BboltDB) OpenWrite() derrors.Error {
	return b.openDB( false)
}

// TODO: think about if it is better return *bolt.DB. It could exist a case where we need to reuse this
// If we decide on it, add this parameter if unsafe operations
func (b * BboltDB) openDB( readOnly bool)  derrors.Error {
	db, err := bolt.Open(b.Path, 0600, &bolt.Options{ReadOnly: readOnly, Timeout: BboltTimeOut * time.Second})
	if err != nil {
		log.Error().Str("Path", b.Path).Msg("error opening bbolt database")
		return derrors.NewInternalError("error opening bbolt database")
	}

	b.ReadOnly = readOnly
	b.DB = db
	return nil
}

func (b * BboltDB) Close() {
	if b.DB != nil {
		b.DB.Close()
	}
	b.DB = nil
}

func (b * BboltDB) CheckConnection() derrors.Error {
	if b.DB == nil {
		err := b.openDB(b.ReadOnly)
		if err != nil {
			log.Error().Str("Path", b.Path).Msg("error opening bbolt database")
			return derrors.NewInternalError("error opening bbolt database")
		}
	}
	return nil
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