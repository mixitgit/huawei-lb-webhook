package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type LoadBalancerAnnotator struct {
	Name     string
	Client   client.Client
	decoder  *admission.Decoder
	LBConfig *LoadBalancerConfig
}

type LoadBalancerConfig struct {
	NamePrefix   string `yaml:"namePrefix"`
	SubnetID     string `yaml:"subnetID"`
	EnterpriseID string `yaml:"enterpriseID"`
	Autocreate   `yaml:"autocreate"`
}

type Autocreate struct {
	LBType              string `yaml:"type" json:"type"`
	BandwidthName       string `yaml:",omitempty" json:"bandwidth_name,omitempty"`
	BandwidthChargemode string `yaml:"bandwidthChargemode,omitempty" json:"bandwidth_chargemode,omitempty"`
	BandwidthSize       int `yaml:"bandwidthSize,omitempty" json:"bandwidth_size,omitempty"`
	BandwidthSharetype  string `yaml:"bandwidthSharetype,omitempty" json:"bandwidth_sharetype,omitempty"`
	EipType             string `yaml:"eipType,omitempty" json:"eip_type,omitempty"`
	Name                string `yaml:",omitempty" json:"name,omitempty"`
}

func (lba *LoadBalancerAnnotator) annotateLoadBalancer(svc *corev1.Service) error {
	if _, e := svc.Annotations["kubernetes.io/elb.class"]; e {
		return fmt.Errorf("Annotation elb.class exists")
	}
	if _, e := svc.Annotations["kubernetes.io/session-affinity-mode"]; e {
		return fmt.Errorf("Annotation kubernetes.io/session-affinity-mode exists")
	}
	if _, e := svc.Annotations["kubernetes.io/elb.subnet-id"]; e {
		return fmt.Errorf("Annotation kubernetes.io/elb.subnet-id exists")
	}
	if _, e := svc.Annotations["kubernetes.io/elb.enterpriseID"]; e {
		return fmt.Errorf("Annotation kubernetes.io/elb.enterpriseID exists")
	}
	if _, e := svc.Annotations["kubernetes.io/elb.autocreate"]; e {
		return fmt.Errorf("Annotation kubernetes.io/autocreate exists")
	}

	svc.Annotations["kubernetes.io/elb.class"] = "union"
	svc.Annotations["kubernetes.io/session-affinity-mode"] = "SOURCE_IP"
	svc.Annotations["kubernetes.io/elb.subnet-id"] = lba.LBConfig.SubnetID
	svc.Annotations["kubernetes.io/elb.enterpriseID"] = lba.LBConfig.EnterpriseID

	lba.LBConfig.Name = fmt.Sprintf("lb-%s-%s-%s", lba.LBConfig.NamePrefix, svc.ObjectMeta.Name, svc.ObjectMeta.Namespace)
	lba.LBConfig.BandwidthName = fmt.Sprintf("bw-%s-%s-%s", lba.LBConfig.NamePrefix, svc.ObjectMeta.Name, svc.ObjectMeta.Namespace)
	marshalledAutocreate, err := json.Marshal(lba.LBConfig.Autocreate)
	if err != nil {
		log.Info("LBAnnotator: cannot marshal")
		return fmt.Errorf("Failed to marshal autocreate,", err)
	}
	log.Info("Generated autocreate:", string(marshalledAutocreate))
	svc.Annotations["kubernetes.io/elb.autocreate"] = string(marshalledAutocreate)

	return nil
}

func (lba *LoadBalancerAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	service := &corev1.Service{}

	err := lba.decoder.Decode(req, service)
	if err != nil {
		log.Info("LBAnnotator: cannot decode")
		return admission.Errored(http.StatusBadRequest, err)
	}

	log.Info(fmt.Sprintf("Got service %s namespace %s type: %s", service.ObjectMeta.Name, service.ObjectMeta.Namespace, service.Spec.Type))

	var shouldAnnotate bool
	if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
		shouldAnnotate = true
	}

	log.Info("Should Annotate: ", shouldAnnotate)

	if shouldAnnotate {
		log.Info("Annotating service...")

		err = lba.annotateLoadBalancer(service)
		if err != nil {
			log.Info("Unable to annotate service: ", err)
		} else {
			log.Info("Service annoted")
		}
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
