#!/bin/bash

cd "$(dirname "$0")" || exit

KUBECONFIG=${KUBECONFIG:=${HOME}/.kube/config} 

if [ ! -f "$KUBECONFIG" ]; then
    echo "Kubeconfig $KUBECONFIG not found!"
    exit 1
fi

if [ -z "$DISK_OFFERING_ID" ]; then
    echo "Variable DISK_OFFERING_ID not set!"
    exit 1
fi

# Create storage class "cloudstack-csi-driver-e2e"
scName="cloudstack-csi-driver-e2e"
sed "s/<disk-offering-id>/${DISK_OFFERING_ID}/" storageclass.yaml | kubectl apply -f -

# Run in parallel when possible (exclude [Feature:.*], [Disruptive] and [Serial]):
./ginkgo -p -progress -v \
       -focus='External.Storage.*csi-cloudstack' \
       -skip='\[Feature:|\[Disruptive\]|\[Serial\]' \
       e2e.test -- \
       -storage.testdriver=testdriver.yaml \
        --kubeconfig="$KUBECONFIG"

# Then run the remaining tests, sequentially:
./ginkgo -progress -v \
       -focus='External.Storage.*csi-cloudstack.*(\[Feature:|\[Disruptive\]|\[Serial\])' \
       e2e.test -- \
       -storage.testdriver=testdriver.yaml \
       --kubeconfig="$KUBECONFIG"

# Delete storage class
kubectl delete storageclasses.storage.k8s.io "${scName}" || echo "No storage class named ${scName}"