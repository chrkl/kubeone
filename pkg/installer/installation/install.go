/*
Copyright 2019 The KubeOne Authors.

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

package installation

import (
	"github.com/pkg/errors"

	"github.com/kubermatic/kubeone/pkg/addons"
	"github.com/kubermatic/kubeone/pkg/certificate"
	"github.com/kubermatic/kubeone/pkg/credentials"
	"github.com/kubermatic/kubeone/pkg/features"
	"github.com/kubermatic/kubeone/pkg/kubeconfig"
	"github.com/kubermatic/kubeone/pkg/state"
	"github.com/kubermatic/kubeone/pkg/task"
	"github.com/kubermatic/kubeone/pkg/templates/externalccm"
	"github.com/kubermatic/kubeone/pkg/templates/machinecontroller"
	"github.com/kubermatic/kubeone/pkg/templates/nodelocaldns"
)

// Install performs all the steps required to install Kubernetes on
// an empty, pristine machine.
func Install(s *state.State) error {
	installSteps := []task.Task{
		{Fn: installPrerequisites, ErrMsg: "failed to install prerequisites"},
		{Fn: generateKubeadm, ErrMsg: "failed to generate kubeadm config files"},
		{Fn: kubeadmCertsOnLeader, ErrMsg: "failed to provision certs and etcd on leader", Retries: 10},
		{Fn: certificate.DownloadCA, ErrMsg: "unable to download ca from leader", Retries: 10},
		{Fn: deployCA, ErrMsg: "unable to deploy ca on nodes", Retries: 10},
		{Fn: kubeadmCertsOnFollower, ErrMsg: "failed to provision certs and etcd on followers", Retries: 10},
		{Fn: initKubernetesLeader, ErrMsg: "failed to init kubernetes on leader", Retries: 10},
		{Fn: joinControlplaneNode, ErrMsg: "unable to join other masters a cluster", Retries: 10},
		{Fn: copyKubeconfig, ErrMsg: "unable to copy kubeconfig to home directory", Retries: 10},
		{Fn: saveKubeconfig, ErrMsg: "unable to save kubeconfig to the local machine", Retries: 10},
		{Fn: kubeconfig.BuildKubernetesClientset, ErrMsg: "unable to build kubernetes clientset", Retries: 10},
		{Fn: nodelocaldns.Deploy, ErrMsg: "unable to deploy nodelocaldns", Retries: 10},
		{Fn: features.Activate, ErrMsg: "unable to activate features", Retries: 10},
		{Fn: ensureCNI, ErrMsg: "failed to install cni plugin", Retries: 10},
		{Fn: patchCoreDNS, ErrMsg: "failed to patch CoreDNS", Retries: 10},
		{Fn: credentials.Ensure, ErrMsg: "unable to ensure credentials secret", Retries: 10},
		{Fn: externalccm.Ensure, ErrMsg: "failed to install external CCM", Retries: 10},
		{Fn: patchCNI, ErrMsg: "failed to patch CNI", Retries: 10},
		{Fn: machinecontroller.Ensure, ErrMsg: "failed to install machine-controller", Retries: 10},
		{Fn: machinecontroller.WaitReady, ErrMsg: "failed to wait for machine-controller", Retries: 10},
		{Fn: createWorkerMachines, ErrMsg: "failed to create worker machines", Retries: 10},
		{Fn: addons.Ensure, ErrMsg: "failed to apply addons", Retries: 10},
	}

	for _, step := range installSteps {
		if err := step.Run(s); err != nil {
			return errors.Wrap(err, step.ErrMsg)
		}
	}

	return nil
}
