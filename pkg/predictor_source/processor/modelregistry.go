package processor

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	"github.com/opendatahub-io/model-registry/pkg/openapi"
	"k8s.io/apimachinery/pkg/types"
)

const (
	modelRegistryBaseUrl        string = "MODEL_REGISTRY_BASE_URL"
	defaultModelRegistryBaseUrl        = "modelregistry-sample.model-registry.svc.cluster.local:8080"
	modelRegistryScheme         string = "MODEL_REGISTRY_SCHEME"
	defaultModelRegistryScheme         = "http"
)

var _ CustomStorageProcessor = (*ModelRegistryProcessor)(nil)

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

// ProcessInferenceServiceStorage implements CustomStorageProcessor.
func (m *ModelRegistryProcessor) ProcessInferenceServiceStorage(
	customUri *url.URL,
	inferenceService *v1beta1.InferenceService,
	nname types.NamespacedName,
	processInferenceServiceStorage func(inferenceService *v1beta1.InferenceService, nname types.NamespacedName) (*string, map[string]string, string, *string, error),
) (secretKey *string, parameters map[string]string, modelPath string, schemaPath *string, err error) {
	tokens := strings.SplitN(strings.TrimPrefix(customUri.Path, "/"), "/", 2)

	if len(tokens) == 0 || len(tokens) > 2 {
		err = fmt.Errorf("invalid model registry URI, use like model-registry://{registeredModelName}/{versionName}/{artifactId}")
		return
	}

	modelName := customUri.Host
	var versionName *string
	var artifactId *string
	if len(tokens) == 1 {
		versionName = &tokens[0]
	}
	if len(tokens) == 2 {
		artifactId = &tokens[1]
	}

	modelArtifact, err1 := m.FindModel(modelName, versionName, artifactId)
	if err1 != nil {
		err = fmt.Errorf("unable to find model %v: %w", modelName, err1)
		return
	}

	transformedInferenceService := inferenceService.DeepCopy()
	transformedInferenceService.Spec.Predictor.Model.StorageURI = modelArtifact.Uri

	// TODO(user): handle all possible models specs

	secretKey, parameters, modelPath, schemaPath, err = processInferenceServiceStorage(transformedInferenceService, nname)
	return
}

// FindModel contact the model registry using Model Registry client to retrieve model artifact details
func (m *ModelRegistryProcessor) FindModel(modelName string, versionName *string, artifactId *string) (*openapi.ModelArtifact, error) {
	// Get RegisteredModel by name
	model, _, err := m.Client.ModelRegistryServiceAPI.FindRegisteredModel(context.Background()).Name(modelName).Execute()
	if err != nil {
		return nil, err
	}

	// Get ModelVersion by name or latest
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

	// Get ModelArtifact by id or latest
	var artifact *openapi.ModelArtifact
	if artifactId != nil {
		artifact, _, err = m.Client.ModelRegistryServiceAPI.GetModelArtifact(context.Background(), *artifactId).Execute()
		if err != nil {
			return nil, err
		}
	} else {
		artifacts, _, err := m.Client.ModelRegistryServiceAPI.GetModelVersionArtifacts(context.Background(), *version.Id).
			OrderBy(openapi.ORDERBYFIELD_CREATE_TIME).
			SortOrder(openapi.SORTORDER_DESC).
			Execute()
		if err != nil {
			return nil, err
		}

		modelArtifacts := []*openapi.ModelArtifact{}
		for _, a := range artifacts.Items {
			if a.ModelArtifact != nil {
				modelArtifacts = append(modelArtifacts, a.ModelArtifact)
			}
		}

		if len(modelArtifacts) == 0 {
			return nil, fmt.Errorf("no model artifacts associated to model version %s", *version.Id)
		}

		artifact = modelArtifacts[0]
	}

	return artifact, nil
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
