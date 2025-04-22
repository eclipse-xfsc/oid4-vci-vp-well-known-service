package git

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/eclipse-xfsc/oid4-vci-vp-library/model/credential"

	ctxPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/ctx"
	logPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/logr"
	serverPkg "github.com/eclipse-xfsc/microservice-core-go/pkg/server"

	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/config"
	"github.com/eclipse-xfsc/oid4-vci-vp-well-known-service/internal/common"
	"github.com/gin-gonic/gin"
	"github.com/madflojo/tasks"
	"gopkg.in/src-d/go-git.v4"
)

type Importer struct {
	config        config.GitConfig
	log           logPkg.Logger
	taskScheduler *tasks.Scheduler
	folder        string
	repo          *git.Repository
	lastError     error
}

const (
	issuerJSON              = "issuer.json"
	credentialsSupportedDir = "credentials"
	cacheDir                = "cache"
)

func NewImporter(config config.GitConfig, logger logPkg.Logger) *Importer {
	return &Importer{
		config:        config,
		folder:        assemblePath(os.TempDir(), cacheDir),
		log:           logger,
		taskScheduler: tasks.New(),
	}
}

func (g *Importer) Start(ctx context.Context, server *serverPkg.Server, _ *common.Environment) error {
	server.Add(func(rg *gin.RouterGroup) {
		rg.Static(g.config.ImagePath, g.folder)
	})

	_, err := g.taskScheduler.Add(&tasks.Task{
		TaskContext: tasks.TaskContext{Context: ctx},
		Interval:    g.config.Interval,
		TaskFunc:    g.checkout,
	})

	if err != nil {
		g.log.Error(err, "Failed to create scheduler for git importer")
		return err
	}

	return nil
}

func (g *Importer) Stop() error {
	g.taskScheduler.Stop()
	return nil
}

func (g *Importer) GotErrors() bool {
	return g.lastError != nil
}

func (g *Importer) GetCredentialIssuerMetadata(ctx context.Context, tenantID string) (*credential.IssuerMetadata, error) {
	issuerPath := assemblePath(g.folder, tenantID)

	issuerData, err := os.ReadFile(assemblePath(issuerPath, issuerJSON))
	if err != nil {
		g.log.Error(err, "failed to read file from disk")
		return nil, err
	}

	var issuer credential.IssuerMetadata
	if err := json.Unmarshal(issuerData, &issuer); err != nil {
		return nil, fmt.Errorf("failed to decode issuer.json: %w", err)
	}

	issuer.CredentialConfigurationsSupported, err = g.collectCredentialsSupported(ctx, assemblePath(issuerPath, credentialsSupportedDir))
	if err != nil {
		return nil, fmt.Errorf("failed to collectCredentialsSupported: %w", err)
	}

	return &issuer, nil
}

func (g *Importer) collectCredentialsSupported(ctx context.Context, path string) (map[string]credential.CredentialConfiguration, error) {
	logger := ctxPkg.GetLogger(ctx)

	files, err := os.ReadDir(path)
	if err != nil {
		logger.Error(err, "Error reading Directory")
		return nil, err
	}

	credentials := make(map[string]credential.CredentialConfiguration)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		data, err := os.ReadFile(assemblePath(path, file.Name()))
		if err != nil {
			continue
		}

		var credential credential.CredentialConfiguration
		if err := json.Unmarshal(data, &credential); err != nil {
			g.log.Error(err, "failed to unmarshal credentials supported")
			continue
		}

		credentials[file.Name()] = credential
	}

	return credentials, nil
}

func (g *Importer) checkout() error {
	g.log.Info("git clone " + g.folder)

	cloneOpt := git.CloneOptions{
		URL:      g.config.Repo,
		Progress: os.Stdout,
	}

	if token := g.config.Token; token != "" {
		cloneOpt.URL = strings.ReplaceAll(cloneOpt.URL, "https://", "https://token:"+token+"@")
	}

	if g.repo == nil {
		gitRepo, err := git.PlainClone(g.folder, false, &cloneOpt)
		if err != nil {
			return err
		}

		g.repo = gitRepo

		return nil
	}

	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}

	if err := w.Pull(&git.PullOptions{RemoteName: "origin"}); err != nil {
		return err
	}

	ref, err := g.repo.Head()
	if err != nil {
		return err
	}

	commit, err := g.repo.CommitObject(ref.Hash())
	if err != nil {
		return err
	}

	g.log.Info("Pulled " + commit.String())

	return nil
}

func assemblePath(paths ...string) string {
	path := ""
	for _, file := range paths {
		path = fmt.Sprintf("%c%s", os.PathSeparator, file)
	}

	return strings.TrimLeft(path, string(os.PathSeparator))
}
