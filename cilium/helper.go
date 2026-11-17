package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cilium/cilium/cilium-cli/clustermesh"
	"github.com/cilium/cilium/cilium-cli/defaults"
	"github.com/cilium/cilium/cilium-cli/k8s"
	"github.com/cilium/cilium/cilium-cli/status"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v4/pkg/action"

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
	// err is the error returned while creating the Kubernetes client. It is
	// reported by resources only when they need to reach the cluster, so that
	// a kubeconfig created in the same apply does not break the plan.
	err error
}

// Ready returns an error if the Kubernetes client is not available.
func (c *CiliumClient) Ready() error {
	if c == nil {
		return fmt.Errorf("unable to connect to kubernetes: the provider is not configured")
	}
	if c.err != nil {
		return fmt.Errorf("unable to connect to kubernetes: %w", c.err)
	}
	return nil
}

func ConcatDefault(text string, d string) string {
	return fmt.Sprintf("%s (Default: `%s`).", text, d)
}

// DefaultCiliumVersion is the Cilium version installed when none is specified.
// renovate: datasource=github-releases depName=cilium/cilium
const DefaultCiliumVersion = "1.20.5"

// tempFiles tracks the temporary files that must outlive a single RPC (e.g. the
// kubeconfig written from config_content) so they can be removed on exit.
var (
	tempFilesMu sync.Mutex
	tempFiles   []string
)

// CleanupTempFiles removes the temporary files created by the provider.
func CleanupTempFiles() {
	tempFilesMu.Lock()
	defer tempFilesMu.Unlock()
	for _, f := range tempFiles {
		os.Remove(f)
	}
	tempFiles = nil
}

// writeTempFile writes content to a new temporary file and returns its path.
// The caller is responsible for removing it.
func writeTempFile(pattern string, content []byte) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func (c *CiliumClient) Wait(ctx context.Context) (err error) {
	var status_params = status.K8sStatusParameters{}
	status_params.Namespace = c.namespace
	status_params.Wait = true
	status_params.WaitDuration = defaults.StatusWaitDuration
	collector, err := status.NewK8sStatusCollector(c.client, status_params)
	if err != nil {
		return err
	}
	_, err = collector.Status(ctx)
	return err
}

func (c *CiliumClient) helmActionConfig() (*action.Configuration, error) {
	// Use the default Helm driver (Kubernetes secret).
	helmDriver := ""
	actionConfig := action.Configuration{}
	if err := actionConfig.Init(c.client.RESTClientGetter, c.namespace, helmDriver); err != nil {
		return nil, err
	}
	return &actionConfig, nil
}

// ReleaseExists returns nil if the Cilium Helm release exists.
func (c *CiliumClient) ReleaseExists() error {
	actionConfig, err := c.helmActionConfig()
	if err != nil {
		return err
	}
	_, err = actionConfig.Releases.Last(c.helm_release)
	return err
}

func (c *CiliumClient) GetHelmValues() (string, error) {
	actionConfig, err := c.helmActionConfig()
	if err != nil {
		return "", err
	}
	vals, err := action.NewGetValues(actionConfig).Run(c.helm_release)
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
	actionConfig, err := c.helmActionConfig()
	if err != nil {
		return "", err
	}
	vals, err := action.NewGetMetadata(actionConfig).Run(c.helm_release)
	if err != nil {
		return "", err
	}

	return vals.AppVersion, nil
}

func (c *CiliumClient) WaitClusterMesh(ctx context.Context) (err error) {
	var params = clustermesh.Parameters{Writer: os.Stdout}
	params.Namespace = c.namespace
	params.Wait = true
	params.WaitDuration = 2 * time.Minute
	cm := clustermesh.NewK8sClusterMesh(c.client, params)
	if _, err := cm.Status(ctx); err != nil {
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

func ValueList(ctx context.Context, l types.List) []string {
	destinationContexts := make([]types.String, 0, len(l.Elements()))
	l.ElementsAs(ctx, &destinationContexts, false)

	d := []string{}
	for _, e := range destinationContexts {
		d = append(d, e.ValueString())
	}
	return d
}
