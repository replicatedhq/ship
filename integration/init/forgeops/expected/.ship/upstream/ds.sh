#!/usr/bin/env bash
# Utility script to check for the backup pvc, and pass it to helm if present.

ARG=""
kubectl get pvc ds-backup && ARG="--set backup.pvcClaimName=ds-backup" 


helm install $ARG $* ds