package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cilium/cilium/cilium-cli/clustermesh"
	"github.com/cilium/cilium/cilium-cli/defaults"
	"github.com/cilium/cilium/cilium-cli/k8s"
	"github.com/cilium/cilium/cilium-cli/status"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var CaAttributeTypes = map[string]attr.Type{
	"crt": types.StringType,
	"key": types.StringType,
}

type CiliumClient struct {
	client       *k8s.Client
	namespace    string
	helm_release string
}

func ConcatDefault(text string, d string) string {
	return fmt.Sprintf("%s (Default: `%s`).", text, d)
}

func (c *CiliumClient) Wait() (err error) {
	var status_params = status.K8sStatusParameters{}
	status_params.Namespace = c.namespace
	status_params.Wait = true
	status_params.WaitDuration = defaults.StatusWaitDuration
	collector, err := status.NewK8sStatusCollector(c.client, status_params)
	if err != nil {
		return err
	}
	_, err = collector.Status(context.Background())
	return err
}

func (c *CiliumClient) GetCurrentRelease() (*release.Release, error) {
	// Use the default Helm driver (Kubernetes secret).
	helmDriver := ""
	actionConfig := action.Configuration{}
	logger := func(format string, v ...interface{}) {}
	if err := actionConfig.Init(c.client.RESTClientGetter, c.namespace, helmDriver, logger); err != nil {
		return nil, err
	}
	currentRelease, err := actionConfig.Releases.Last(c.helm_release)
	if err != nil {
		return nil, err
	}
	return currentRelease, nil
}

func (c *CiliumClient) GetHelmValues() (string, error) {
	helmDriver := ""
	actionConfig := action.Configuration{}
	logger := func(format string, v ...interface{}) {}
	if err := actionConfig.Init(c.client.RESTClientGetter, c.namespace, helmDriver, logger); err != nil {
		return "", err
	}
	client := action.NewGetValues(&actionConfig)

	vals, err := client.Run(c.helm_release)
	if err != nil {
		return "", err
	}

	yaml, err := yaml.Marshal(vals)
	if err != nil {
		return "", err
	}
	return string(yaml), nil
}

func (c *CiliumClient) GetCA(ctx context.Context) (map[string]attr.Value, error) {
	k8sClient := c.client
	s, err := k8sClient.GetSecret(ctx, c.namespace, "cilium-ca", metav1.GetOptions{})
	if err != nil {
		return map[string]attr.Value{}, err
	}
	ca := map[string]attr.Value{
		"key": types.StringValue(base64.StdEncoding.EncodeToString(s.Data["ca.key"])),
		"crt": types.StringValue(base64.StdEncoding.EncodeToString(s.Data["ca.crt"])),
	}

	return ca, nil
}

func (c *CiliumClient) GetMetadata() (string, error) {
	helmDriver := ""
	actionConfig := action.Configuration{}
	logger := func(format string, v ...interface{}) {}
	if err := actionConfig.Init(c.client.RESTClientGetter, c.namespace, helmDriver, logger); err != nil {
		return "", err
	}
	client := action.NewGetMetadata(&actionConfig)

	vals, err := client.Run(c.helm_release)
	if err != nil {
		return "", err
	}

	return vals.AppVersion, nil
}

func (c *CiliumClient) WaitClusterMesh() (err error) {
	var params = clustermesh.Parameters{Writer: os.Stdout}
	params.Namespace = c.namespace
	params.Wait = true
	params.WaitDuration = 2 * time.Minute
	cm := clustermesh.NewK8sClusterMesh(c.client, params)
	if _, err := cm.Status(context.Background()); err != nil {
		return err
	}
	return nil
}

func (c *CiliumClient) CheckDaemonsetStatus(ctx context.Context, namespace, daemonset string) error {
	k8sClient := c.client
	d, _ := k8sClient.GetDaemonSet(ctx, namespace, daemonset, metav1.GetOptions{})
	if d == nil {
		return nil
	}

	if d.Status.NumberReady != 0 {
		return fmt.Errorf("replicas count is not zero")
	}

	return nil
}

func (c *CiliumClient) CheckDaemonsetAvailability(ctx context.Context, namespace, daemonset string) error {
	k8sClient := c.client
	d, err := k8sClient.GetDaemonSet(ctx, namespace, daemonset, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if d == nil {
		return fmt.Errorf("daemonset is not available")
	}

	return nil
}
