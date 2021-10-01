package azure

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/tfvc"
)

type AzureInterface interface {
	Azure() *Azure
	Connect()                           // Подключение к Azure DevOps
	TfvcClientConnection() error        // для Repository.Open()
	ListOfProjects() ([]*string, error) // Получаем список проектов

	GetChangesets(nameOfProject string) ([]*int, error)              // Получает все id ченджсетов проекта
	GetChangesetChanges(id *int, project string) (*ChangeSet, error) // получает все изминения для конкретного changeSet
	ChangedRows(currentFilePath, version string) (int, int, error)   // Принимает ссылки на разные версии файлов возвращает Добавленные и Удаленные строки
}

type ChangeSet struct {
	ProjectName string
	Id          int
	Author      string
	Email       string
	AddedRows   int
	DeletedRows int
	Date        time.Time
	Message     string
	Hash        string
}

type Azure struct {
	Config     *Config
	Connection *azuredevops.Connection
	TfvcClient tfvc.Client
}

func NewAzure(conf *Config) AzureInterface {
	return &Azure{
		Config: conf,
	}
}

func (a *Azure) Azure() *Azure {
	return a
}

func (a *Azure) Connect() {
	organizationUrl := a.Config.OrganizationUrl
	personalAccessToken := a.Config.Token

	connection := azuredevops.NewPatConnection(organizationUrl, personalAccessToken)
	a.Connection = connection
}

func (a *Azure) TfvcClientConnection() error {
	tfvcClient, err := tfvc.NewClient(a.Config.Context, a.Connection)
	if err != nil {
		return err
	}
	a.TfvcClient = tfvcClient
	return nil
}

func (a *Azure) ListOfProjects() ([]*string, error) {
	coreClient, err := core.NewClient(a.Config.Context, a.Connection)
	if err != nil {
		return nil, err
	}

	resp, err := coreClient.GetProjects(a.Config.Context, core.GetProjectsArgs{})
	if err != nil {
		return nil, err
	}
	projectNames := []*string{}
	for _, project := range resp.Value {
		projectNames = append(projectNames, project.Name)
	}
	return projectNames, nil
}

func (a *Azure) GetChangesets(nameOfProject string) ([]*int, error) {
	changeSets, err := a.TfvcClient.GetChangesets(a.Config.Context, tfvc.GetChangesetsArgs{Project: &nameOfProject})
	if err != nil {
		return nil, err
	}
	changeSetIDs := []*int{}
	for _, v := range *changeSets {
		changeSetIDs = append(changeSetIDs, v.ChangesetId)
	}
	return changeSetIDs, nil
}

func (a *Azure) GetChangesetChanges(id *int, project string) (*ChangeSet, error) {
	changes, err := a.TfvcClient.GetChangeset(a.Config.Context, tfvc.GetChangesetArgs{Id: id, Project: &project})
	if err != nil {
		return nil, err
	}
	messg := ""
	if changes.Comment != nil {
		messg = *changes.Comment
	}
	getChanges, err := a.TfvcClient.GetChangesetChanges(a.Config.Context, tfvc.GetChangesetChangesArgs{Id: id})
	if err != nil {
		return nil, err
	}

	//получаем кол-во добавленных и удаленных строк
	addedRows := 0
	deletedRows := 0
	for _, v := range getChanges.Value {
		if v.Item.(map[string]interface{})["isFolder"] != nil {
			if v.Item.(map[string]interface{})["isFolder"].(bool) {
				continue
			}
		}

		path := v.Item.(map[string]interface{})["path"].(string)
		if isImage(path) {
			continue
		}

		ar, dr, err := a.ChangedRows(path, fmt.Sprint(v.Item.(map[string]interface{})["version"]))
		if err != nil {
			return nil, err
		}
		addedRows += ar
		deletedRows += dr
	}

	commit := &ChangeSet{
		ProjectName: project,
		Id:          *id,
		Author:      *changes.Author.DisplayName,
		Email:       *changes.Author.UniqueName,
		Date:        changes.CreatedDate.Time,
		AddedRows:   addedRows,
		DeletedRows: deletedRows,
		Message:     messg,
	}
	return commit, nil
}

func (a *Azure) ChangedRows(currentFilePath, version string) (int, int, error) {
	// Берем текущую версию файла
	currentItemContent, err := a.TfvcClient.GetItemContent(a.Config.Context, tfvc.GetItemContentArgs{Path: &currentFilePath,
		VersionDescriptor: &git.TfvcVersionDescriptor{Version: &version}})
	if err != nil {
		return 0, 0, err
	}
	currentFile, err := io.ReadAll(currentItemContent)
	if err != nil {
		return 0, 0, err
	}

	// Берем редыдущую версию файла
	previousFileContent, err := a.TfvcClient.GetItemContent(a.Config.Context, tfvc.GetItemContentArgs{Path: &currentFilePath,
		VersionDescriptor: &git.TfvcVersionDescriptor{Version: &version, VersionOption: &git.TfvcVersionOptionValues.Previous}})
	if err != nil { //если нет прошлой версии считаем кол-во строк в текущем файле
		arrTransformStrings := strings.Split(string(currentFile), "\n")
		return len(arrTransformStrings), 0, nil
	}

	previousFile, err := io.ReadAll(previousFileContent)
	if err != nil {
		return 0, 0, err
	}

	// Считаем добаленные и удаленные строки
	addedRows, deletedRows := Diff(string(previousFile), string(currentFile))
	return addedRows, deletedRows, nil
}

func isImage(path string) bool {
	if strings.Contains(path, ".jpg") ||
		strings.Contains(path, ".jpeg") ||
		strings.Contains(path, ".png") {
		return true
	}
	return false
}
