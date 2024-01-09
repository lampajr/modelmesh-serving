package processor

import (
	"net/url"

	"github.com/kserve/kserve/pkg/apis/serving/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

// CustomStorageProcessor defines a generic custom URI processor
type CustomStorageProcessor interface {
	ProcessInferenceServiceStorage(
		customUri *url.URL,
		inferenceService *v1beta1.InferenceService,
		nname types.NamespacedName,
		processInferenceServiceStorage func(inferenceService *v1beta1.InferenceService, nname types.NamespacedName) (*string, map[string]string, string, *string, error),
	) (secretKey *string, parameters map[string]string, modelPath string, schemaPath *string, err error)
}
