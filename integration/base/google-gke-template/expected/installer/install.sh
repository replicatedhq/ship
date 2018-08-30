#!/bin/bash
echo "run:"
echo "terraform apply -f terraform/complex_cluster.tf"
echo "kubectl apply -f kube.yaml --kubeconfig terraform/kubeconfig_complex-cluster"
