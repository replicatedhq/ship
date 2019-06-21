#!/bin/bash
echo "applying kuberentes configs"
kubectl apply -f ./k8s/* -n 
