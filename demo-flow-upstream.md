- Spin up 3 workload clusters
- Get kubeconfig for last one
- Install HelmChartProxy
- Add label to two clusters
- Change version to see revision
- Scale control plane on one cluster and show that controller updates chart
- Delete out of band and wait for controller to recreate
- Delete CRD and show that controller removes helm charts