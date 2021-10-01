package cli_metrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-marathon-team-3/pkg/tfsmetrics"
	"go-marathon-team-3/pkg/tfsmetrics/azure"
	"go-marathon-team-3/pkg/tfsmetrics/exporter"
	"go-marathon-team-3/pkg/tfsmetrics/repointerface"
	"go-marathon-team-3/pkg/tfsmetrics/store"
	"strconv"
	"sync"
	"time"

	"github.com/urfave/cli/v2"

	"io/ioutil"
	"os"
	"path"
)

type cliSettings struct {
	CacheEnabled bool `json:"cache-enabled"`
	ExporterPort int  `json:"exporter-port"`
}

func CreateMetricsApp(prjPath *string) *cli.App {
	app := cli.NewApp()
	app.Name = "cli-metrics"
	app.Usage = "CLI для взаимодействия с библиотекой"
	app.EnableBashCompletion = true
	app.Version = "1.0"
	app.Authors = []*cli.Author{
		{Name: "Андрей Назаренко"},
		{Name: "Артем Богданов"},
		{Name: "Алексей Вологдин"},
	}
	settingsPath := path.Join(*prjPath, "configs/cli-settings.json")
	settings, _ := ReadSettingsFile(&settingsPath)
	localStore, _ := store.NewStore()
	var url, token, cache string
	var author, project string
	var port int
	app.Commands = []*cli.Command{
		{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "установка параметров, необходимых для подключения к Azure (подробнее см. cli-metrics config --help)",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "organization-url",
					Aliases:     []string{"url", "u"},
					Usage:       "url для подключения к Azure",
					Destination: &url,
				},
				&cli.StringFlag{
					Name:        "access-token",
					Aliases:     []string{"token", "t"},
					Usage:       "personal access token для подключения к Azure",
					Destination: &token,
				},
				&cli.StringFlag{
					Name:        "cache-enabled",
					Aliases:     []string{"cache", "c"},
					Usage:       "логический флаг следует ли использовать кеш при работе программы",
					Destination: &cache,
				},
				&cli.IntFlag{
					Name:        "exporter-port",
					Aliases:     []string{"port", "p"},
					Usage:       "номер порта, на котором запускается экспортер",
					Value:       8080,
					Destination: &port,
				},
			},
			Action: func(c *cli.Context) error {
				configPath := path.Join(*prjPath, "configs/config.json")
				config, err := ReadConfigFile(&configPath)
				if err != nil {
					return err
				}
				if url != "" {
					config.OrganizationUrl = url
				}
				if token != "" {
					config.Token = token
				}
				if cache == "true" {
					settings.CacheEnabled = true
				} else if cache != "" {
					settings.CacheEnabled = false
				}
				if port != settings.ExporterPort {
					if port < 1024 || port > 65535 {
						return errors.New("Введите порт в диапазоне от 1024 до 65535!")
					} else {
						settings.ExporterPort = port
					}
				}
				err = WriteConfigFile(&configPath, config)
				if err != nil {
					return err
				}
				err = WriteSettingsFile(&settingsPath, settings)
				fmt.Printf("Текущая конфигурация:\nURL: %s\nToken: %s\nCacheEnabled: %t\nExporterPort: %d\n",
					config.OrganizationUrl, config.Token, settings.CacheEnabled, settings.ExporterPort)
				return err
			},
		},
		{
			Name:    "getmetrics",
			Aliases: []string{"gm"},
			Usage:   "вывод на экран данных метрики по конкретному автору или по проекту",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "author",
					Aliases:     []string{"a"},
					Usage:       "данные метрики по конкретному автору",
					Destination: &author,
				},
				&cli.StringFlag{
					Name:        "project",
					Aliases:     []string{"p"},
					Usage:       "данные метрики по конкретному проекту",
					Destination: &project,
				},
			},
			Action: func(c *cli.Context) error {
				if author == "" && project == "" {
					return errors.New("Пожалуйста, укажите автора или название проекта.")
				}
				var err error
				azureClient, err := connect(prjPath)
				if err != nil {
					return err
				}
				settings, _ := ReadSettingsFile(&settingsPath)
				exp := exporter.NewExporter()
				if project != "" {
					commits := tfsmetrics.NewCommitCollection(project, azureClient, settings.CacheEnabled, localStore)
					err = commits.Open()
					if err != nil {
						return err
					}
					iter, err := commits.GetCommitIterator()
					if err != nil {
						return err
					}
					data := exp.GetDataByProject(iter)
					fmt.Printf("Данные метрики по проекту '%s':\n", project)
					printByProject(&data)
					fmt.Println()
				}
				if author != "" {
					data := make(map[string]*exporter.ByAuthor)
					projectNames, err := azureClient.ListOfProjects()
					if err != nil {
						return err
					}
					for _, prj := range projectNames {
						commits := tfsmetrics.NewCommitCollection(*prj, azureClient, settings.CacheEnabled, localStore)
						err = commits.Open()
						if err != nil {
							return err
						}
						iter, err := commits.GetCommitIterator()
						if err != nil {
							return err
						}
						data = exp.GetDataByAuthor(iter, author, *prj)
					}
					fmt.Printf("Данные метрики по автору '%s':\n", author)
					printByAuthor(&data)
					fmt.Println()
				}
				return nil
			},
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "вывод на экран названий всех проектов в репозитории",
			Action: func(context *cli.Context) error {
				var err error
				azureClient, err := connect(prjPath)
				if err != nil {
					return err
				}
				projectNames, err := azureClient.ListOfProjects()
				if err != nil {
					return err
				}
				fmt.Println("Доступны следующие проекты:")
				for ind, project := range projectNames {
					fmt.Printf("%d) %s\n", ind+1, *project)
				}
				return nil
			},
		},
		{
			Name:    "log",
			Aliases: []string{"l"},
			Usage:   "получение информации обо всех коммитах",
			Action: func(context *cli.Context) error {
				var err error
				prjName := context.Args().Get(0)
				azureClient, err := connect(prjPath)
				settings, _ := ReadSettingsFile(&settingsPath)
				if err != nil {
					return err
				}
				projectNames, err := azureClient.ListOfProjects()
				if err != nil {
					return err
				}
				if prjName == "" {
					fmt.Println("Название проекта не было указано, информация по коммитам будет выведена по всем проектам:")
					for _, project := range projectNames {
						_ = processProject(project, &azureClient, settings.CacheEnabled, &localStore)
					}
				} else {
					for _, project := range projectNames {
						if *project == prjName {
							err = processProject(project, &azureClient, settings.CacheEnabled, &localStore)
							if err != nil {
								return err
							}
						}
					}
				}
				return nil
			},
		},
		{
			Name:    "start-exporter",
			Aliases: []string{"s"},
			Usage:   "запуск экспортера (для запуска в фоне введите: nohup cli-metrics start-exporter &)",
			Action: func(context *cli.Context) error {
				var err error
				azureClient, err := connect(prjPath)
				if err != nil {
					return err
				}
				settings, _ := ReadSettingsFile(&settingsPath)
				projectNames, err := azureClient.ListOfProjects()
				if err != nil {
					return err
				}
				exp := exporter.NewExporter()
				for _, project := range projectNames {
					commits := tfsmetrics.NewCommitCollection(*project, azureClient, settings.CacheEnabled, localStore)
					err = commits.Open()
					if err != nil {
						return err
					}
					iter, err := commits.GetCommitIterator()
					if err != nil {
						return err
					}
					exp.PrometheusMetrics(iter, *project)
				}
				fmt.Printf("Метрики доступны по адресу http://localhost:%d/metrics\n", settings.ExporterPort)
				wg := sync.WaitGroup{}
				serv := exporter.NewPrometheusServer(&wg, time.Second*5)
				serv.Start(":" + strconv.Itoa(settings.ExporterPort))
				wg.Wait()
				return nil
			},
		},
	}
	azure.NewConfig()
	return app
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func ReadConfigFile(filePath *string) (config *azure.Config, err error) {
	config = azure.NewConfig()
	ex, _ := exists(*filePath)
	if !ex {
		output, _ := os.Create(*filePath)
		defer output.Close()
		jsonEncoder := json.NewEncoder(output)
		err = jsonEncoder.Encode(config)
	}
	data, err := ioutil.ReadFile(*filePath)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &config)
	return
}

func WriteConfigFile(filePath *string, config *azure.Config) error {
	output, err := os.Create(*filePath)
	if err != nil {
		return err
	}
	defer output.Close()
	jsonEncoder := json.NewEncoder(output)
	err = jsonEncoder.Encode(config)
	return err
}

func WriteSettingsFile(filePath *string, settings *cliSettings) error {
	output, err := os.Create(*filePath)
	if err != nil {
		return err
	}
	defer output.Close()
	jsonEncoder := json.NewEncoder(output)
	err = jsonEncoder.Encode(settings)
	return err
}

func ReadSettingsFile(filePath *string) (settings *cliSettings, err error) {
	settings = &cliSettings{CacheEnabled: true, ExporterPort: 8080}
	ex, _ := exists(*filePath)
	if !ex {
		output, _ := os.Create(*filePath)
		defer output.Close()
		jsonEncoder := json.NewEncoder(output)
		err = jsonEncoder.Encode(settings)
		return
	}
	data, err := ioutil.ReadFile(*filePath)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &settings)
	return
}

func printFullCommit(commit *repointerface.Commit) {
	fmt.Printf("Автор: %s <%s>\n", commit.Author, commit.Email)
	fmt.Printf("Дата: %s\n", commit.Date.Format("2006-01-02 15:04:05"))
	fmt.Printf("%d строк добавлено и %d строк удалено\n", commit.AddedRows, commit.DeletedRows)
	fmt.Printf("Сообщение:\n\n\t%s\n\n", commit.Message)
	fmt.Println("---------------------------------------------------------------------------------------------------")
}

func printProjectName(name *string) {
	fmt.Println("---------------------------------------------------------------------------------------------------")
	fmt.Printf("\t\t\tПроект %s:\n", *name)
	fmt.Println("---------------------------------------------------------------------------------------------------")
	fmt.Printf("\n\n")
}

func processProject(project *string, azureClient *azure.AzureInterface, cacheEnabled bool, localStore *store.Store) error {
	printProjectName(project)
	commits := tfsmetrics.NewCommitCollection(*project, *azureClient, cacheEnabled, *localStore)
	err := commits.Open()
	if err != nil {
		return err
	}
	iter, err := commits.GetCommitIterator()
	if err != nil {
		return err
	}
	for commit, err := iter.Next(); err == nil; commit, err = iter.Next() {
		printFullCommit(commit)
	}
	return nil
}

func connect(prjPath *string) (azure.AzureInterface, error) {
	filePath := path.Join(*prjPath, "configs/config.json")
	config, err := ReadConfigFile(&filePath)
	if err != nil {
		return nil, err
	}
	if config.OrganizationUrl == "" && config.Token == "" {
		return nil, errors.New("отсутствуют параметры подключения (cli-metrics config)")
	} else if config.OrganizationUrl == "" {
		return nil, errors.New("отсутствует url подключения (cli-metrics config --url)")
	} else if config.Token == "" {
		return nil, errors.New("отсутствует token подключения (cli-metrics config --token)")
	}
	azureClient := azure.NewAzure(config)
	azureClient.Connect()
	err = azureClient.TfvcClientConnection()
	if err != nil {
		return nil, err
	}
	return azureClient, err
}

func printByAuthor(byauthor *map[string]*exporter.ByAuthor) {
	for project, stats := range *byauthor {
		fmt.Printf("Проект: %s\n", project)
		fmt.Printf("\tКоличество коммитов: %d\n\tКоличество добавленных строк %d\n\tКоличество удаленных строк %d\n",
			(*stats).Commits, (*stats).AddedRows, (*stats).DeletedRows)
	}

}

func printByProject(byproject *map[string]*exporter.ByProject) {
	for author, stats := range *byproject {
		fmt.Printf("Автор: %s\n", author)
		fmt.Printf("\tКоличество коммитов: %d\n\tКоличество добавленных строк %d\n\tКоличество удаленных строк %d\n",
			(*stats).Commits, (*stats).AddedRows, (*stats).DeletedRows)
	}

}
