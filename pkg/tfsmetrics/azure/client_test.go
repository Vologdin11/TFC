package azure

import (
	"errors"
	"go-marathon-team-3/pkg/tfsmetrics/mock"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/tfvc"
	"github.com/microsoft/azure-devops-go-api/azuredevops/webapi"
	"github.com/stretchr/testify/assert"
)

func TestAzure_GetChangesetChanges(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockedClient := mock.NewMockClient(ctrl)

	conf := NewConfig()
	azure := Azure{
		Config:     conf,
		TfvcClient: mockedClient,
	}
	cs := ChangeSet{
		ProjectName: "project",
		Id:          1,
		Author:      "Ivan",
		Email:       "example@example.com",
		AddedRows:   2,
		DeletedRows: 1,
		Date:        time.Now(),
		Message:     "hello world",
		Hash:        "",
	}

	// правильная работа, без ощибки
	mockedClient.
		EXPECT().
		GetChangeset(azure.Config.Context, tfvc.GetChangesetArgs{Id: &cs.Id, Project: &cs.ProjectName}).
		Return(&git.TfvcChangeset{
			Author:      &webapi.IdentityRef{DisplayName: &cs.Author, UniqueName: &cs.Email},
			CreatedDate: &azuredevops.Time{Time: cs.Date},
			Comment:     &cs.Message,
		}, nil)

	version := "1"
	currentFilePath := "currentFilePath"
	mockedClient.
		EXPECT().
		GetChangesetChanges(azure.Config.Context, tfvc.GetChangesetChangesArgs{Id: &cs.Id}).
		Return(&tfvc.GetChangesetChangesResponseValue{Value: []git.TfvcChange{
			{Item: map[string]interface{}{"isFolder": true}},
			{Item: map[string]interface{}{"path": currentFilePath, "version": version}},
			{Item: map[string]interface{}{"path": "image.jpg", "version": version}},
		}}, nil)

	currentFileContent := io.ReadCloser(io.NopCloser(strings.NewReader("current file content\n row")))
	previousFileContent := io.ReadCloser(io.NopCloser(strings.NewReader("previous file content")))

	// две версии
	mockedClient.
		EXPECT().
		GetItemContent(azure.Config.Context, tfvc.GetItemContentArgs{Path: &currentFilePath,
			VersionDescriptor: &git.TfvcVersionDescriptor{Version: &version}}).
		Return(currentFileContent, nil)

	mockedClient.
		EXPECT().
		GetItemContent(azure.Config.Context, tfvc.GetItemContentArgs{Path: &currentFilePath,
			VersionDescriptor: &git.TfvcVersionDescriptor{Version: &version, VersionOption: &git.TfvcVersionOptionValues.Previous}}).
		Return(previousFileContent, nil)

	changeSet, err := azure.GetChangesetChanges(&cs.Id, cs.ProjectName)
	assert.NoError(t, err)
	assert.Equal(t, &cs, changeSet)

	// azure возвращает ошибку
	cs.Id += 2
	mockedClient.
		EXPECT().
		GetChangeset(azure.Config.Context, tfvc.GetChangesetArgs{Id: &cs.Id, Project: &cs.ProjectName}).
		Return(nil, errors.New("error"))

	changeSet, err = azure.GetChangesetChanges(&cs.Id, cs.ProjectName)
	assert.Error(t, err)
	assert.Nil(t, changeSet)
}

func TestAzure_ChangedRows(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockedClient := mock.NewMockClient(ctrl)

	conf := NewConfig()
	azure := Azure{
		Config:     conf,
		TfvcClient: mockedClient,
	}
	currentFilePath := "currentFilePath"
	version := "1"
	currentFileContent := io.ReadCloser(io.NopCloser(strings.NewReader("current file content\n row")))
	previousFileContent := io.ReadCloser(io.NopCloser(strings.NewReader("previous file content")))

	// две версии
	mockedClient.
		EXPECT().
		GetItemContent(azure.Config.Context, tfvc.GetItemContentArgs{Path: &currentFilePath,
			VersionDescriptor: &git.TfvcVersionDescriptor{Version: &version}}).
		Return(currentFileContent, nil)

	mockedClient.
		EXPECT().
		GetItemContent(azure.Config.Context, tfvc.GetItemContentArgs{Path: &currentFilePath,
			VersionDescriptor: &git.TfvcVersionDescriptor{Version: &version, VersionOption: &git.TfvcVersionOptionValues.Previous}}).
		Return(previousFileContent, nil)
	addedRows, deletedRows, err := azure.ChangedRows(currentFilePath, version)
	assert.NoError(t, err)
	assert.Equal(t, 2, addedRows)
	assert.Equal(t, 1, deletedRows)

	// новый файл одна версия
	currentFileContent = io.ReadCloser(io.NopCloser(strings.NewReader("current file content\n row")))
	mockedClient.
		EXPECT().
		GetItemContent(azure.Config.Context, tfvc.GetItemContentArgs{Path: &currentFilePath,
			VersionDescriptor: &git.TfvcVersionDescriptor{Version: &version}}).
		Return(currentFileContent, nil)

	mockedClient.
		EXPECT().
		GetItemContent(azure.Config.Context, tfvc.GetItemContentArgs{Path: &currentFilePath,
			VersionDescriptor: &git.TfvcVersionDescriptor{Version: &version, VersionOption: &git.TfvcVersionOptionValues.Previous}}).
		Return(nil, errors.New("error"))

	addedRows, deletedRows, err = azure.ChangedRows(currentFilePath, version)
	assert.NoError(t, err)
	assert.Equal(t, 2, addedRows)
	assert.Equal(t, 0, deletedRows)
}
