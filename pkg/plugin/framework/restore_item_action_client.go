/*
Copyright 2017, 2019 the Velero contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"encoding/json"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	proto "github.com/heptio/velero/pkg/plugin/generated"
	"github.com/heptio/velero/pkg/plugin/velero"
)

var _ velero.RestoreItemAction = &RestoreItemActionGRPCClient{}

// NewRestoreItemActionPlugin constructs a RestoreItemActionPlugin.
func NewRestoreItemActionPlugin(options ...PluginOption) *RestoreItemActionPlugin {
	return &RestoreItemActionPlugin{
		pluginBase: newPluginBase(options...),
	}
}

// RestoreItemActionGRPCClient implements the backup/ItemAction interface and uses a
// gRPC client to make calls to the plugin server.
type RestoreItemActionGRPCClient struct {
	*clientBase
	grpcClient proto.RestoreItemActionClient
}

func newRestoreItemActionGRPCClient(base *clientBase, clientConn *grpc.ClientConn) interface{} {
	return &RestoreItemActionGRPCClient{
		clientBase: base,
		grpcClient: proto.NewRestoreItemActionClient(clientConn),
	}
}

func (c *RestoreItemActionGRPCClient) AppliesTo() (velero.ResourceSelector, error) {
	res, err := c.grpcClient.AppliesTo(context.Background(), &proto.AppliesToRequest{Plugin: c.plugin})
	if err != nil {
		return velero.ResourceSelector{}, err
	}

	return velero.ResourceSelector{
		IncludedNamespaces: res.IncludedNamespaces,
		ExcludedNamespaces: res.ExcludedNamespaces,
		IncludedResources:  res.IncludedResources,
		ExcludedResources:  res.ExcludedResources,
		LabelSelector:      res.Selector,
	}, nil
}

func (c *RestoreItemActionGRPCClient) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	itemJSON, err := json.Marshal(input.Item.UnstructuredContent())
	if err != nil {
		return nil, err
	}

	itemFromBackupJSON, err := json.Marshal(input.ItemFromBackup.UnstructuredContent())
	if err != nil {
		return nil, err
	}

	restoreJSON, err := json.Marshal(input.Restore)
	if err != nil {
		return nil, err
	}

	req := &proto.RestoreExecuteRequest{
		Plugin:         c.plugin,
		Item:           itemJSON,
		ItemFromBackup: itemFromBackupJSON,
		Restore:        restoreJSON,
	}

	res, err := c.grpcClient.Execute(context.Background(), req)
	if err != nil {
		return nil, err
	}

	var updatedItem unstructured.Unstructured
	if err := json.Unmarshal(res.Item, &updatedItem); err != nil {
		return nil, err
	}

	var warning error
	if res.Warning != "" {
		warning = errors.New(res.Warning)
	}

	return &velero.RestoreItemActionExecuteOutput{
		UpdatedItem: &updatedItem,
		Warning:     warning,
	}, nil
}
