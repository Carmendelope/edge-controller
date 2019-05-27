/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package asset

import (
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/satori/go.uuid"
	"time"
)

func CreateTestAgentOpRequest(assetID string) * entities.AgentOpRequest{
	params := make(map[string]string, 0)
	params["p1"]="v1"
	params["p2"]="v2"
	return &entities.AgentOpRequest{
		Created:          time.Now().Unix(),
		OrganizationId:   uuid.NewV4().String(),
		EdgeControllerId: uuid.NewV4().String(),
		AssetId:          assetID,
		Operation:        "test",
		Plugin:           "test",
		Params:           params,
	}
}

func CreateTestAgentOpResponse(assetID string) * entities.AgentOpResponse{
	return &entities.AgentOpResponse{
		Created:          time.Now().Unix(),
		OrganizationId:   uuid.NewV4().String(),
		EdgeControllerId: uuid.NewV4().String(),
		AssetId:          assetID,
		OperationId:      uuid.NewV4().String(),
		Timestamp:        time.Now().Unix(),
		Status:           grpc_inventory_manager_go.AgentOpStatus_SUCCESS.String(),
		Info:             "",
	}
}

func CreateTestAgentStartInfo(assetID string) * entities.AgentStartInfo{
	return &entities.AgentStartInfo{
		Created: time.Now().Unix(),
		AssetId: assetID,
		Ip:      "1.1.1.1",
	}
}

func CreateTestAgentJoinInfo(assetID string) * entities.AgentJoinInfo{
	return &entities.AgentJoinInfo{
		Created: time.Now().Unix(),
		AssetId: assetID,
		Token:   uuid.NewV4().String(),
	}
}

func RegisterAsset(assetID string, provider Provider) string{
	toAdd := entities.AgentJoinInfo{
		Created: time.Now().Unix(),
		AssetId: assetID,
		Token:   uuid.NewV4().String(),
	}
	err := provider.AddManagedAsset(toAdd)
	gomega.Expect(err).To(gomega.Succeed())
	return toAdd.Token
}


func RunTest(provider Provider){

	ginkgo.BeforeEach(func() {
		provider.Clear()
	})

	ginkgo.Context("Operation Requests", func(){
		ginkgo.It("should be able to add a new pending operation", func(){
			assetID := uuid.NewV4().String()
			RegisterAsset(assetID, provider)
			toAdd := CreateTestAgentOpRequest(assetID)
			err := provider.AddPendingOperation(*toAdd)
			gomega.Expect(err).To(gomega.Succeed())
		})
		ginkgo.It("should be able to list operations", func(){
		    numOps := 10
		    assetID := uuid.NewV4().String()
			RegisterAsset(assetID, provider)
		    for i := 0; i < numOps; i ++{
				toAdd := CreateTestAgentOpRequest(assetID)
				err := provider.AddPendingOperation(*toAdd)
				gomega.Expect(err).To(gomega.Succeed())
			}

		    retrieved, err := provider.GetPendingOperations(assetID, false)
		    gomega.Expect(err).To(gomega.Succeed())
		    gomega.Expect(len(retrieved)).Should(gomega.Equal(numOps))

			retrievedRemoving, err := provider.GetPendingOperations(assetID, true)
			gomega.Expect(err).To(gomega.Succeed())
			gomega.Expect(len(retrievedRemoving)).Should(gomega.Equal(numOps))

			retrieved, err = provider.GetPendingOperations(assetID, false)
			gomega.Expect(err).To(gomega.Succeed())
			gomega.Expect(len(retrieved)).Should(gomega.Equal(0))
		})
	})

	ginkgo.Context("Operation responses", func(){
		ginkgo.It("should be able to add an operation response", func(){
			assetID := uuid.NewV4().String()
			RegisterAsset(assetID, provider)
			toAdd := CreateTestAgentOpResponse(assetID)
			err := provider.AddOpResponse(*toAdd)
			gomega.Expect(err).To(gomega.Succeed())
		})
		ginkgo.It("should be able to get pending operations", func(){
			numOps := 10
			assetID := uuid.NewV4().String()
			RegisterAsset(assetID, provider)
			for i := 0; i < numOps; i ++{
				toAdd := CreateTestAgentOpResponse(assetID)
				err := provider.AddOpResponse(*toAdd)
				gomega.Expect(err).To(gomega.Succeed())
			}

			retrieved, err := provider.GetPendingOpResponses(false)
			gomega.Expect(err).To(gomega.Succeed())
			gomega.Expect(len(retrieved)).Should(gomega.Equal(numOps))

			retrievedRemoving, err := provider.GetPendingOpResponses(true)
			gomega.Expect(err).To(gomega.Succeed())
			gomega.Expect(len(retrievedRemoving)).Should(gomega.Equal(numOps))

			retrieved, err = provider.GetPendingOpResponses(false)
			gomega.Expect(err).To(gomega.Succeed())
			gomega.Expect(len(retrieved)).Should(gomega.Equal(0))
		})
	})

	ginkgo.Context("Agent start", func(){
		ginkgo.It("should be able to add an agent start message", func(){
			assetID := uuid.NewV4().String()
			RegisterAsset(assetID, provider)
			toAdd := CreateTestAgentStartInfo(assetID)
			err := provider.AddAgentStart(*toAdd)
			gomega.Expect(err).To(gomega.Succeed())
		})
		ginkgo.It("should be able to retrieve pending start messages", func(){
			numOps := 10
			for i := 0; i < numOps; i ++{
				assetID := uuid.NewV4().String()
				RegisterAsset(assetID, provider)
				toAdd := CreateTestAgentStartInfo(assetID)
				err := provider.AddAgentStart(*toAdd)
				gomega.Expect(err).To(gomega.Succeed())
			}

			retrieved, err := provider.GetPendingAgentStart(false)
			gomega.Expect(err).To(gomega.Succeed())
			gomega.Expect(len(retrieved)).Should(gomega.Equal(numOps))

			retrievedRemoving, err := provider.GetPendingAgentStart(true)
			gomega.Expect(err).To(gomega.Succeed())
			gomega.Expect(len(retrievedRemoving)).Should(gomega.Equal(numOps))

			retrieved, err = provider.GetPendingAgentStart(false)
			gomega.Expect(err).To(gomega.Succeed())
			gomega.Expect(len(retrieved)).Should(gomega.Equal(0))
		})
	})

	ginkgo.Context("Managed assets", func(){
		ginkgo.It("should be able to add a managed asset", func(){
			assetID := uuid.NewV4().String()
			toAdd := CreateTestAgentJoinInfo(assetID)
			err := provider.AddManagedAsset(*toAdd)
			gomega.Expect(err).To(gomega.Succeed())
		})
		ginkgo.It("should be able to retrieve the agent by token", func(){
			assetID := uuid.NewV4().String()
			toAdd := CreateTestAgentJoinInfo(assetID)
			err := provider.AddManagedAsset(*toAdd)
			gomega.Expect(err).To(gomega.Succeed())
			info, err := provider.GetAssetByToken(toAdd.Token)
			gomega.Expect(err).To(gomega.Succeed())
			gomega.Expect(info.AssetId).Should(gomega.Equal(assetID))
		})
		ginkgo.It("should be able to remove an asset", func(){
			assetID := uuid.NewV4().String()
			toAdd := CreateTestAgentJoinInfo(assetID)
			err := provider.AddManagedAsset(*toAdd)
			gomega.Expect(err).To(gomega.Succeed())
			err = provider.RemoveManagedAsset(assetID)
			gomega.Expect(err).To(gomega.Succeed())
			_, err = provider.GetAssetByToken(toAdd.Token)
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("Join tokens", func(){
		ginkgo.It("should be able add a join token", func(){
		    token := uuid.NewV4().String()
		    _, err := provider.AddJoinToken(token)
		    gomega.Expect(err).To(gomega.Succeed())
		})
		ginkgo.It("should be able to check a join token", func(){
			token := uuid.NewV4().String()
			_, err := provider.AddJoinToken(token)
			gomega.Expect(err).To(gomega.Succeed())
			result, err := provider.CheckJoinJoin(token)
			gomega.Expect(err).To(gomega.Succeed())
			gomega.Expect(result).Should(gomega.BeTrue())
		})
	})
}