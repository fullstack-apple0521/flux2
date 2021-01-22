/*
Copyright 2020 The Flux authors

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

package main

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/fluxcd/flux2/internal/utils"
	"github.com/fluxcd/pkg/apis/meta"

	"github.com/spf13/cobra"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
)

var getHelmReleaseCmd = &cobra.Command{
	Use:     "helmreleases",
	Aliases: []string{"hr"},
	Short:   "Get HelmRelease statuses",
	Long:    "The get helmreleases command prints the statuses of the resources.",
	Example: `  # List all Helm releases and their status
  flux get helmreleases
`,
	RunE: getHelmReleaseCmdRun,
}

func init() {
	getCmd.AddCommand(getHelmReleaseCmd)
}

func getHelmReleaseCmdRun(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), rootArgs.timeout)
	defer cancel()

	kubeClient, err := utils.KubeClient(rootArgs.kubeconfig, rootArgs.kubecontext)
	if err != nil {
		return err
	}

	var listOpts []client.ListOption
	if !getArgs.allNamespaces {
		listOpts = append(listOpts, client.InNamespace(rootArgs.namespace))
	}
	var list helmv2.HelmReleaseList
	err = kubeClient.List(ctx, &list, listOpts...)
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		logger.Failuref("no releases found in %s namespace", rootArgs.namespace)
		return nil
	}

	header := []string{"Name", "Ready", "Message", "Revision", "Suspended"}
	if getArgs.allNamespaces {
		header = append([]string{"Namespace"}, header...)
	}
	var rows [][]string
	for _, helmRelease := range list.Items {
		row := []string{}
		if c := apimeta.FindStatusCondition(helmRelease.Status.Conditions, meta.ReadyCondition); c != nil {
			row = []string{
				helmRelease.GetName(),
				string(c.Status),
				c.Message,
				helmRelease.Status.LastAppliedRevision,
				strings.Title(strconv.FormatBool(helmRelease.Spec.Suspend)),
			}
		} else {
			row = []string{
				helmRelease.GetName(),
				string(metav1.ConditionFalse),
				"waiting to be reconciled",
				helmRelease.Status.LastAppliedRevision,
				strings.Title(strconv.FormatBool(helmRelease.Spec.Suspend)),
			}
		}
		if getArgs.allNamespaces {
			row = append([]string{helmRelease.Namespace}, row...)
		}
		rows = append(rows, row)
	}
	utils.PrintTable(os.Stdout, header, rows)
	return nil
}
