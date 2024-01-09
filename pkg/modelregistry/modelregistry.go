package modelregistry

import (
	"context"
	"fmt"
	"os"

	"github.com/opendatahub-io/model-registry/pkg/openapi"
)

const (
	modelRegistryBaseUrl        string = "MODEL_REGISTRY_BASE_URL"
	defaultModelRegistryBaseUrl        = "modelregistry-sample.model-registry.svc.cluster.local:8080"
	modelRegistryScheme         string = "MODEL_REGISTRY_SCHEME"
	defaultModelRegistryScheme         = "http"
)

type ModelRegistryProcessor struct {
	Client *openapi.APIClient
}

func NewModelRegistryProcessorFromEnv() *ModelRegistryProcessor {
	cfg := openapi.NewConfiguration()
	cfg.Host = getEnvString(modelRegistryBaseUrl, defaultModelRegistryBaseUrl)
	cfg.Scheme = getEnvString(modelRegistryScheme, defaultModelRegistryScheme)

	return newModelRegistryProcessor(cfg)
}

func newModelRegistryProcessor(cfg *openapi.Configuration) *ModelRegistryProcessor {
	client := openapi.NewAPIClient(cfg)

	return &ModelRegistryProcessor{
		Client: client,
	}
}

// FindModel contact the model registry using Model Registry client to retrieve model artifact details
func (m *ModelRegistryProcessor) FindModel(modelName string, versionName *string, artifactId *string) (*openapi.ModelArtifact, error) {
	// Fetch the registered model
	model, _, err := m.Client.ModelRegistryServiceAPI.FindRegisteredModel(context.Background()).Name(modelName).Execute()
	if err != nil {
		return nil, err
	}

	// Fetch model version by name or latest if not specified
	var version *openapi.ModelVersion
	if versionName != nil {
		version, _, err = m.Client.ModelRegistryServiceAPI.FindModelVersion(context.Background()).Name(*versionName).ParentResourceID(*model.Id).Execute()
		if err != nil {
			return nil, err
		}
	} else {
		versions, _, err := m.Client.ModelRegistryServiceAPI.GetRegisteredModelVersions(context.Background(), *model.Id).
			OrderBy(openapi.ORDERBYFIELD_CREATE_TIME).
			SortOrder(openapi.SORTORDER_DESC).
			Execute()
		if err != nil {
			return nil, err
		}

		if versions.Size == 0 {
			return nil, fmt.Errorf("no versions associated to registered model %s", modelName)
		}
		version = &versions.Items[0]
	}

	artifacts, _, err := m.Client.ModelRegistryServiceAPI.GetModelVersionArtifacts(context.Background(), *version.Id).
		OrderBy(openapi.ORDERBYFIELD_CREATE_TIME).
		SortOrder(openapi.SORTORDER_DESC).
		Execute()
	if err != nil {
		return nil, err
	}

	if artifacts.Size == 0 {
		return nil, fmt.Errorf("no artifacts associated to model version %s", *version.Id)
	}

	modelArtifact := artifacts.Items[0].ModelArtifact
	if modelArtifact == nil {
		return nil, fmt.Errorf("no model artifact found for model version %s", *version.Id)
	}

	return modelArtifact, nil
}

// Returns the string value of environment variable "key" or the default value
// if "key" is not set. Note if the environment variable is set to an empty
// string, this will return an empty string, not defaultValue.
func getEnvString(key string, defaultValue string) string {
	if val, found := os.LookupEnv(key); found {
		return val
	}
	return defaultValue
}
