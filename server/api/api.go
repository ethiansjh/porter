package api

import (
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"

	"github.com/gorilla/sessions"
	"github.com/porter-dev/porter/internal/helm"
	"github.com/porter-dev/porter/internal/kubernetes"
	lr "github.com/porter-dev/porter/internal/logger"
	"github.com/porter-dev/porter/internal/repository"
	"helm.sh/helm/v3/pkg/storage"
)

// TestAgents are the k8s agents used for testing
type TestAgents struct {
	HelmAgent             *helm.Agent
	HelmTestStorageDriver *storage.Storage
	K8sAgent              *kubernetes.Agent
}

// App represents an API instance with handler methods attached, a DB connection
// and a logger instance
type App struct {
	logger     *lr.Logger
	repo       *repository.Repository
	validator  *validator.Validate
	store      sessions.Store
	translator *ut.Translator
	cookieName string
	testing    bool
	TestAgents *TestAgents
}

// New returns a new App instance
func New(
	logger *lr.Logger,
	repo *repository.Repository,
	validator *validator.Validate,
	store sessions.Store,
	cookieName string,
	testing bool,
) *App {
	// for now, will just support the english translator from the
	// validator/translations package
	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")

	var testAgents *TestAgents = nil

	if testing {
		memStorage := helm.StorageMap["memory"](nil, nil, "")

		testAgents = &TestAgents{
			HelmAgent:             helm.GetAgentTesting(&helm.Form{}, nil, logger),
			HelmTestStorageDriver: memStorage,
			K8sAgent:              kubernetes.GetAgentTesting(),
		}
	}

	return &App{
		logger:     logger,
		repo:       repo,
		validator:  validator,
		store:      store,
		translator: &trans,
		cookieName: cookieName,
		testing:    testing,
		TestAgents: testAgents,
	}
}

// Logger returns the logger instance in use by App
func (app *App) Logger() *lr.Logger {
	return app.logger
}
