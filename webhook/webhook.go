package hook

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type LoadBalancerAnnotator struct {
	Name               string
	Client             client.Client
	decoder            *admission.Decoder
}

func (lba *LoadBalancerAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	service := &corev1.Service{}

	err := lba.decoder.Decode(req, service)
	if err != nil {
		log.Info("LBAnnotator: cannot decode")
		return admission.Errored(http.StatusBadRequest, err)
	}

	var shouldAnnotate bool
	if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
		shouldAnnotate = true
	}

	log.Info("Should Annotate: ", shouldAnnotate, service.Spec.Type)

	if shouldAnnotate {
		log.Info("Annotating service...")

		service.Annotations["test"] = "true"

		log.Info("Service annoted")
	} else {
		log.Info("Annotation not needed")
	}

	marshalledService, err := json.Marshal(service)

	if err != nil {
		log.Info("LBAnnotator: cannot marshal")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshalledService)
}

// InjectDecoder injects the decoder.
func (lba *LoadBalancerAnnotator) InjectDecoder(d *admission.Decoder) error {
	lba.decoder = d
	return nil
}
