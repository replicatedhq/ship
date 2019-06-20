#!/bin/bash
set -e
kubectl get namespace  | wc -l
echo "? Namespace Created"
