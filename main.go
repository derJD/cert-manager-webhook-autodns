package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/klog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	cmd.RunWebhookServer(GroupName,
		&autoDNSProviderSolver{},
	)
}

type autoDNSProviderSolver struct {
	client *kubernetes.Clientset
}

type autoDNSProviderConfig struct {
	Zone       string `json:"zone"`
	NameServer string `json:"nameserver"`
	Context    string `json:"context"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	URL        string `json:"url"`
}

type AutoDNSData struct {
	Origin             string                      `json:"origin"`
	ResourceRecord     []AutoDNSResourceRecordData `json:"resourceRecord,omitempty"`
	ResourceRecordsAdd []AutoDNSResourceRecordData `json:"resourceRecordsAdd,omitempty"`
	ResourceRecordsRem []AutoDNSResourceRecordData `json:"resourceRecordsRem,omitempty"`
}

type AutoDNSResourceRecordData struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
	Pref  int64  `json:"pref,omitempty"`
	TTL   int64  `json:"ttl,omitempty"`
}

func (c *autoDNSProviderSolver) Name() string {
	return "autoDNS"
}

func (c *autoDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}

	if cfg.Zone == "" {
		cfg.Zone = ch.ResolvedZone
	}
	jsonData, err := json.Marshal(AutoDNSData{
		Origin: ch.ResolvedZone,
		ResourceRecordsAdd: []AutoDNSResourceRecordData{
			{
				Name:  ch.ResolvedFQDN,
				Value: ch.Key,
				TTL:   60,
				Type:  "TXT",
			},
		},
	})
	if err != nil {
		return err
	}

	return callApi("PATCH", jsonData, cfg)
}

func (c *autoDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}

	if cfg.Zone == "" {
		cfg.Zone = ch.ResolvedZone
	}
	jsonData, err := json.Marshal(AutoDNSData{
		Origin: ch.ResolvedZone,
		ResourceRecordsRem: []AutoDNSResourceRecordData{
			{
				Name:  ch.ResolvedFQDN,
				Value: ch.Key,
				TTL:   60,
				Type:  "TXT",
			},
		},
	})
	if err != nil {
		return err
	}

	return callApi("PATCH", jsonData, cfg)
}

func (c *autoDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	c.client = cl

	return nil
}

func loadConfig(cfgJSON *extapi.JSON) (autoDNSProviderConfig, error) {
	cfg := autoDNSProviderConfig{}
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

func callApi(method string, body []byte, config autoDNSProviderConfig) error {
	url := config.URL + "/zone/" + config.Zone + "/" + config.NameServer
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("unable to execute request %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Domainrobot-Context", config.Context)
	req.SetBasicAuth(config.Username, config.Password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			klog.Fatal(err)
		}
	}()

	//respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	text := "Error calling API status: " + resp.Status + " url: " + url + " method: " + method
	klog.Error(text)
	return errors.New(text)
}
