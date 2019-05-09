package asset

import (
	"encoding/json"
	"fmt"
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/database"
	"github.com/nalej/grpc-utils/pkg/conversions"
	bolt "go.etcd.io/bbolt"
	"sync"
	"time"
)

const (
	assetsByAssetIDBucket 	= "assetsByAssetIDBucket"
	assetsByTokenBucket 	= "assetsByTokenBucket"
	pendingOpsBucket 		= "pendingOpsBucket"
	pendingResultBucket 	= "pendingResultBucket"
	joinTokenBucket 		= "joinTokenBucket"
	agentStartBucket 		= "agentStartBucket"
)

type BboltAssetProvider struct {
	// Mutex for managing provider access.
	sync.Mutex
	database.BboltDB
}

func NewBboltAssetProvider(databasePath string) * BboltAssetProvider{
	return &BboltAssetProvider{
		BboltDB : database.BboltDB{
			Path: databasePath,
		},
	}
}


func (b *BboltAssetProvider) AddPendingOperation(op entities.AgentOpRequest) derrors.Error {
	b.Lock()
	defer b.Unlock()

	obj := make([]entities.AgentOpRequest, 0)

	err := b.OpenWrite(b.Path)
	defer b.Close()

	if err != nil {
		return conversions.ToDerror(err)
	}

	newErr :=  b.DB.Update(func(tx *bolt.Tx) error {

		key := []byte(op.AssetId)


		// check if asset is managed by this EIC, if not exists -> return an error
		bkAsset, err := tx.CreateBucketIfNotExists([]byte(assetsByAssetIDBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", assetsByAssetIDBucket))
		}
		asset := bkAsset.Get(key)
		if asset == nil {
			return derrors.NewFailedPreconditionError("asset is not managed by this EIC").WithParams(op.AssetId)
		}

		// Get pendings operations to add this new one
		bk, err := tx.CreateBucketIfNotExists([]byte(pendingOpsBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", pendingOpsBucket))
		}

		res := bk.Get([]byte (op.AssetId))

		if res != nil {
			if err := json.Unmarshal(res, &obj); err != nil {
				return derrors.NewInternalError("error creating object")
			}
		}
			obj = append(obj, op)

			toAddBytes, err := json.Marshal(obj)
			if err != nil {
				return conversions.ToDerror(err)
			}
			if err := bk.Put([]byte (op.AssetId), toAddBytes); err != nil {
				return derrors.NewInternalError("Cannot add new element")
			}


		return nil

	})

	if newErr != nil {
		return conversions.ToDerror(newErr)
	}

	return nil
}

func (b *BboltAssetProvider) GetPendingOperations(assetID string, removeEntries bool) ([]entities.AgentOpRequest, derrors.Error) {
	b.Lock()
	defer b.Unlock()

	openErr := b.OpenWrite(b.Path)
	defer b.Close()

	result := make([]entities.AgentOpRequest, 0)

	if openErr != nil {
		return result, openErr
	}

	err := b.DB.Update(func(tx *bolt.Tx) error {

		key := []byte(assetID)

		// check if asset is managed by this EIC, if not exists -> return an error
		bkAsset, err := tx.CreateBucketIfNotExists([]byte(assetsByAssetIDBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", assetsByAssetIDBucket))
		}
		asset := bkAsset.Get(key)
		if asset == nil {
			return derrors.NewFailedPreconditionError("asset is not managed by this EIC").WithParams(assetID)
		}

		// Get pending operations
		bk, err := tx.CreateBucketIfNotExists([]byte(pendingOpsBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", pendingOpsBucket))
		}

		res := bk.Get(key)

		if res == nil {
			return nil
		}

		// convert to []entities.AgentOpRequest
		if err := json.Unmarshal(res, &result); err != nil {
			return derrors.NewInternalError("error creating object")
		}

		// if remove if applicable
		if removeEntries{
			if err := bk.Delete([]byte(assetID)); err != nil {
				return derrors.NewInternalError(fmt.Sprintf("Failed to delete '%s': %v", assetID, err))
			}
		}

		return nil
	})

	if err != nil {
		return result, conversions.ToDerror(err)
	}

	return result, nil

}

// AddPendingOperationResult stores a pending operation for an agent.
func (b *BboltAssetProvider) AddOpResponse(op entities.AgentOpResponse) derrors.Error{
	b.Lock()
	defer b.Unlock()

	toAddBytes, err := json.Marshal(op)
	if err != nil {
		return conversions.ToDerror(err)
	}

	openErr := b.OpenWrite(b.Path)
	defer b.Close()

	if openErr != nil {
		return  openErr
	}

	err =  b.DB.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(pendingResultBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", pendingResultBucket))
		}

		key := []byte(op.OperationId)
		// check if already exists
		res := bk.Get(key)

		exists := res != nil
		if exists {
			return derrors.NewAlreadyExistsError("operation result already registered").WithParams(op.OperationId)
		}

		if err := bk.Put(key, toAddBytes); err != nil {
			return derrors.NewInternalError("Cannot registry operation result")
		}
		return nil
	})

	return nil
}

// GetPendingOpResponses retrieves the list of pending operation responses
func (b *BboltAssetProvider) GetPendingOpResponses(removeEntries bool)([]entities.AgentOpResponse, derrors.Error){

	b.Lock()
	defer b.Unlock()

	openErr := b.OpenWrite(b.Path)
	defer b.Close()

	result := make([]entities.AgentOpResponse, 0)

	if openErr != nil {
		return result, openErr
	}

	err := b.DB.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(pendingResultBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", pendingResultBucket))
		}

		bk.ForEach(func(k, v []byte) error {
			var response entities.AgentOpResponse
			if err := json.Unmarshal(v, &response); err != nil {
				return derrors.NewInternalError("error creating object")
			}
			result = append(result, response)

			return nil
		})

		if removeEntries{
			for _, res := range result {
				if err := bk.Delete([]byte(res.OperationId)); err != nil {
					return derrors.NewInternalError(fmt.Sprintf("Failed to delete '%s': %v", res.OperationId, err))
				}
			}
		}

		return nil
	})

	if err != nil {
		return result, conversions.ToDerror(err)
	}

	return result, nil
}

// AddAgentStart stores a pending message with the agent start information.
func (b *BboltAssetProvider) AddAgentStart(op entities.AgentStartInfo) derrors.Error{

	b.Lock()
	defer b.Unlock()

	toAddBytes, err := json.Marshal(op)
	if err != nil {
		return conversions.ToDerror(err)
	}

	openErr := b.OpenWrite(b.Path)
	defer b.Close()
	if openErr != nil {
		return  openErr
	}

	err =  b.DB.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(agentStartBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", agentStartBucket))
		}

		key := []byte(op.AssetId)


		if err := bk.Put(key, toAddBytes); err != nil {
			return derrors.NewInternalError("Cannot add agent")
		}
		return nil
	})

	if err != nil {
		return conversions.ToDerror(err)
	}

	return nil
}

// GetPendingAgetStart retrieves the list of Agent start operations that need to be send
func (b *BboltAssetProvider) GetPendingAgentStart(removeEntries bool) ([]entities.AgentStartInfo, derrors.Error){

	b.Lock()
	defer b.Unlock()

	openErr := b.OpenWrite(b.Path)
	defer b.Close()

	result := make([]entities.AgentStartInfo, 0)

	if openErr != nil {
		return result, openErr
	}

	err := b.DB.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(agentStartBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", agentStartBucket))
		}

		bk.ForEach(func(k, v []byte) error {
			var response entities.AgentStartInfo
			if err := json.Unmarshal(v, &response); err != nil {
				return derrors.NewInternalError("error creating object")
			}
			result = append(result, response)

			return nil
		})

		if removeEntries{
			for _, res := range result {
				if err := bk.Delete([]byte(res.AssetId)); err != nil {
					return derrors.NewInternalError(fmt.Sprintf("Failed to delete '%s': %v", res.AssetId, err))
				}
			}
		}

		return nil
	})

	if err != nil {
		return result, conversions.ToDerror(err)
	}

	return result, nil
}


func (b *BboltAssetProvider) AddManagedAsset(asset entities.AgentJoinInfo) derrors.Error {
	b.Lock()
	defer b.Unlock()

	err := b.OpenWrite(b.Path)
	defer b.Close()

	if err != nil {
		return conversions.ToDerror(err)
	}

	newErr :=  b.DB.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(assetsByAssetIDBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", assetsByAssetIDBucket))
		}

		res := bk.Get([]byte (asset.AssetId))

		if res != nil {
			return derrors.NewAlreadyExistsError("asset already managed by this EIC").WithParams(asset.AssetId)
		}

		toAddBytes, err := json.Marshal(asset)
		if err != nil {
			return conversions.ToDerror(err)
		}
		if err := bk.Put([]byte (asset.AssetId), toAddBytes); err != nil {
			return derrors.NewInternalError("Cannot add new element")
		}

		bkToken, err := tx.CreateBucketIfNotExists([]byte(assetsByTokenBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", assetsByTokenBucket))
		}
		if err := bkToken.Put([]byte (asset.Token), toAddBytes); err != nil {
			return derrors.NewInternalError("Cannot add new element")
		}

		return nil

	})

	if newErr != nil {
		return conversions.ToDerror(newErr)
	}

	return nil
}

func (b *BboltAssetProvider) RemoveManagedAsset(assetID string) derrors.Error {

	b.Lock()
	defer b.Unlock()

	err := b.OpenWrite(b.Path)
	defer b.Close()

	if err != nil {
		return conversions.ToDerror(err)
	}

	newErr :=  b.DB.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(assetsByAssetIDBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", assetsByAssetIDBucket))
		}

		res := bk.Get([]byte (assetID))

		var asset entities.AgentJoinInfo

		if res == nil {
			return derrors.NewFailedPreconditionError("asset is not managed by this EIC").WithParams(assetID)
		}

		if err := json.Unmarshal(res, &asset); err != nil {
			return derrors.NewInternalError("error creating object")
		}


		bkToken, err := tx.CreateBucketIfNotExists([]byte(assetsByTokenBucket))
		if bkToken == nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", assetsByTokenBucket))
		}

		// delete assetsByAssetIDToken
		if err := bkToken.Delete([]byte(asset.Token)); err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to delete '%s': %v", asset.Token, err))
		}

		// delete assetsByAssetIDBucket
		if err := bk.Delete([]byte(assetID)); err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to delete '%s': %v", assetID, err))
		}

		return nil

	})

	if newErr != nil {
		return conversions.ToDerror(newErr)
	}

	return nil
}

func (b *BboltAssetProvider) GetAssetByToken(token string) (*entities.AgentJoinInfo, derrors.Error) {

	b.Lock()
	defer b.Unlock()

	errOpen := b.OpenWrite(b.Path)
	defer b.Close()

	if errOpen != nil {
		return nil, conversions.ToDerror(errOpen)
	}

	var result entities.AgentJoinInfo

	err := b.DB.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(assetsByTokenBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", assetsByTokenBucket))
		}

		res := bk.Get([]byte(token))

		if res == nil {
			return derrors.NewNotFoundError("cannot get the element")
		}

		if err := json.Unmarshal(res, &result); err != nil {
			return derrors.NewInternalError("error creating object")
		}

		return nil
	})

	if err != nil {
		return nil, conversions.ToDerror(err)
	}

	return &result, nil
}

// AddJoinToken adds a new join token for agents
func (b *BboltAssetProvider) AddJoinToken(joinToken string) derrors.Error{

	b.Lock()
	defer b.Unlock()

	toAddBytes, err := json.Marshal(time.Now().Add(AgentJoinTokenTTL).Unix())
	if err != nil {
		return conversions.ToDerror(err)
	}

	openErr := b.OpenWrite(b.Path)
	defer b.Close()
	if openErr != nil {
		return  openErr
	}

	err =  b.DB.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(joinTokenBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", joinTokenBucket))
		}

		key := []byte(joinToken)

		if err := bk.Put(key, toAddBytes); err != nil {
			return derrors.NewInternalError("Cannot add join token")
		}
		return nil
	})

	if err != nil {
		return conversions.ToDerror(err)
	}

	return nil
}

// CheckJoinJoin checks if a join token is valid
func (b *BboltAssetProvider) CheckJoinJoin(joinToken string) (bool, derrors.Error){
	b.Lock()
	defer b.Unlock()

	errOpen := b.OpenWrite(b.Path)
	defer b.Close()

	if errOpen != nil {
		return false, conversions.ToDerror(errOpen)
	}

	check := false

	err := b.DB.Update(func(tx *bolt.Tx) error {
		var result int64

		bk, err := tx.CreateBucketIfNotExists([]byte(joinTokenBucket))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", joinTokenBucket))
		}

		res := bk.Get([]byte(joinToken))

		if res != nil {

			if err := json.Unmarshal(res, &result); err != nil {
				return derrors.NewInternalError("error creating object")
			}
			if result >= time.Now().Unix() {
				check = true
			}else{
				// Expire the token
				if err := bk.Delete([]byte(joinToken)); err != nil {
					return derrors.NewInternalError(fmt.Sprintf("Failed to delete '%s': %v", joinToken, err))
				}
			}
		}

		return nil
	})

	if err != nil {
		return false, conversions.ToDerror(err)
	}

	return check, nil
}

func (b *BboltAssetProvider) clear(table string) derrors.Error{

	b.Lock()
	defer b.Unlock()

	errOpen := b.OpenWrite(b.Path)
	defer b.Close()

	if errOpen != nil {
		return conversions.ToDerror(errOpen)
	}

	err := b.DB.Update(func(tx *bolt.Tx) error {
		bk, err := tx.CreateBucketIfNotExists([]byte(table))
		if err != nil {
			return derrors.NewInternalError(fmt.Sprintf("Failed to get bucket '%s'", table))
		}

		bk.ForEach(func(k, v []byte) error {
			if err := bk.Delete([]byte(k)); err != nil {
				return derrors.NewInternalError(fmt.Sprintf("Failed to delete '%s': %v", k, err))
			}
			return nil
		})

		return nil
	})

	return conversions.ToDerror(err)
}

// Clear all elements
func (b *BboltAssetProvider) Clear() derrors.Error{

	b.clear(assetsByAssetIDBucket)
	b.clear(assetsByTokenBucket)
	b.clear(pendingOpsBucket)
	b.clear(pendingResultBucket)
	b.clear(joinTokenBucket)
	b.clear(agentStartBucket)

	return nil
}