package main

import (
	// "crypto/sha256"
	"crypto/sha256"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	hook "github.com/mixitgit/huawei-lb-webhook/webhook"
	"gopkg.in/yaml.v2"

	// "gopkg.in/yaml.v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var log = logf.Log.WithName("huawei-lb-webhook")

type HookParamters struct {
	certDir  string
	lbConfig string
	port     int
}

func visit(files *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		*files = append(*files, path)
		return nil
	}
}

func loadConfig(configFile string) (*hook.LoadBalancerConfig, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	glog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

	var cfg hook.LoadBalancerConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func main() {
	var params HookParamters

	flag.IntVar(&params.port, "port", 8443, "Wehbook port")
	flag.StringVar(&params.certDir, "certDir", "/certs/", "Wehbook certificate folder")
	flag.StringVar(&params.lbConfig, "lbConfig", "/etc/webhook/config/config.yaml", "Wehbook lb config")
	flag.Parse()

	logf.SetLogger(zap.New())
	entryLog := log.WithName("entrypoint")

	// Setup a Manager
	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	lbConfig, err := loadConfig(params.lbConfig)
	if err != nil {
		entryLog.Error(err, "failed to read loadbalancer config")
		os.Exit(1)
	}

	// Setup webhooks
	entryLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()

	hookServer.Port = params.port
	hookServer.CertDir = params.certDir

	entryLog.Info("registering webhooks to the webhook server")
	hookServer.Register("/mutate", &webhook.Admission{Handler: &hook.LoadBalancerAnnotator{Name: "Huawei", Client: mgr.GetClient(), LBConfig: lbConfig}})

	entryLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}
