#!/bin/bash
echo "run:"
echo "terraform apply -f new/new_vpc.tf"
echo "kubectl apply -f kube.yaml --kubeconfig new/kubeconfig_new-vpc-cluster"
