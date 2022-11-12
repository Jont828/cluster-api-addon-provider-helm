/*
Copyright 2022.

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

package internal

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/external"
	kcpv1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	addonsv1alpha1 "cluster-api-addon-provider-helm/api/v1alpha1"
)

func initializeBuiltins(ctx context.Context, c ctrlClient.Client, references []corev1.ObjectReference, spec addonsv1alpha1.HelmChartProxySpec, cluster *clusterv1.Cluster) (map[string]interface{}, error) {
	log := ctrl.LoggerFrom(ctx)

	// ref := corev1.ObjectReference{
	// 	APIVersion: cluster.APIVersion,
	// 	Kind:       cluster.Kind,
	// 	Namespace:  cluster.Namespace,
	// 	Name:       cluster.Name,
	// }

	valueLookUp := make(map[string]interface{})

	for _, ref := range references {
		log.V(2).Info("Getting object for reference", "ref", ref)
		obj, err := external.Get(ctx, c, &ref, cluster.Namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get object %s", ref.Name)
		}
		valueLookUp[ref.Kind] = obj.Object
	}
	// kubeadmControlPlane := &kcpv1.KubeadmControlPlane{}
	// key := types.NamespacedName{
	// 	Name:      cluster.Spec.ControlPlaneRef.Name,
	// 	Namespace: cluster.Spec.ControlPlaneRef.Namespace,
	// }
	// err := c.Get(ctx, key, kubeadmControlPlane)
	// if err != nil {
	// 	if apierrors.IsNotFound(err) {
	// 		log.V(2).Info("kubeadm control plane not found", "cluster", cluster.Name, "namespace", cluster.Namespace)
	// 	} else {
	// 		return nil, errors.Wrapf(err, "failed to get kubeadm control plane %s", key)
	// 	}
	// }

	// builtInTypes := BuiltinTypes{
	// 	Cluster:            cluster,
	// 	ControlPlane:       kubeadmControlPlane,
	// 	MachineDeployments: map[string]clusterv1.MachineDeployment{},
	// 	MachineSets:        map[string]clusterv1.MachineSet{},
	// 	Machines:           map[string]clusterv1.Machine{},
	// }

	return valueLookUp, nil
}

type BuiltinTypes struct {
	Cluster            *clusterv1.Cluster
	ControlPlane       *kcpv1.KubeadmControlPlane
	MachineDeployments map[string]clusterv1.MachineDeployment
	MachineSets        map[string]clusterv1.MachineSet
	Machines           map[string]clusterv1.Machine
}

/*
	1. Validate that the JSON path template will work on an unstructured Cluster object.
	2. Test to see if we can define a function .GetByReference that wraps external.Get() and if we can call it from the template like so:
		{{ (call .GetByReference (.Cluster.infrastructureRef)).metadata.name }}
	3. Look into referencing an object by a unique type, i.e. AzureCluster since there can only be one
	4. Look into defining provider-generic keys like InfraCluster that get loaded with JSON from external.Get()
*/

func ParseValues(ctx context.Context, c ctrlClient.Client, spec addonsv1alpha1.HelmChartProxySpec, cluster *clusterv1.Cluster) (string, error) {
	log := ctrl.LoggerFrom(ctx)

	log.V(2).Info("Rendering templating in values:", "values", spec.ValuesTemplate)
	references := []corev1.ObjectReference{
		{
			APIVersion: cluster.APIVersion,
			Kind:       cluster.Kind,
			Namespace:  cluster.Namespace,
			Name:       cluster.Name,
		},
	}

	valueLookUp, err := initializeBuiltins(ctx, c, references, spec, cluster)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(spec.ChartName + "-" + cluster.GetName()).
		Funcs(sprig.TxtFuncMap()).
		Parse(spec.ValuesTemplate)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer

	if err := tmpl.Execute(&buffer, valueLookUp); err != nil {
		return "", errors.Wrapf(err, "error executing template string '%s' on cluster '%s'", spec.ValuesTemplate, cluster.GetName())
	}
	expandedTemplate := buffer.String()
	log.V(2).Info("Expanded values to", "result", expandedTemplate)

	return expandedTemplate, nil
}

func ValueMapToArray(valueMap map[string]string) []string {
	valueArray := make([]string, 0, len(valueMap))
	for k, v := range valueMap {
		valueArray = append(valueArray, fmt.Sprintf("%s=%s", k, v))
	}

	return valueArray
}
