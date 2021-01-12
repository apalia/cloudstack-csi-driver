# e2e tests

1. Deploy a Kubernetes cluster on Apache CloudStack;

1. Deploy the cloudstack-csi-driver;

1. Set environment variables:

   | Name               | Description                                                                                              | Required / Default behaviour                  |
   | ------------------ | -------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
   | `DISK_OFFERING_ID` | ID of the CloudStack disk offering to be used in e2e tests. Must accept custom sizes.                    | **REQUIRED**                                  |
   | `KUBECONFIG`       | Path to your `kubeconfig` file                                                                           | Optional - Defaults to `${HOME}/.kube/config` |
   | `KUBE_SSH_USER`    | Username to use to connect to cluster nodes via SSH                                                      | Optional - Defaults to `${USER}`.             |
   | `KUBE_SSH_KEY`     | Path of the SSH key to use to connect to cluster nodes via SSH - may be absolute or relative to `~/.ssh` | Optional - Defaults to `id_rsa`               |
   | `KUBE_SSH_BASTION` | Address (`host:port`) of a bastion host to use to connect to cluster nodes via SSH                       | Optional - Direct connection if not set       |

1. From the root of this repository, execute tests with:

   ```
   make test-e2e
   ```
