## Steps to setting up a GCP hosted runner from scratch

1. Create a disk using opensuse.
```shell
gcloud compute disks create leap-15-4-x86-64 --project=<project-name> --type=pd-standard \
 --size=30GB --zone=<GCP_ZONE> --image=opensuse-leap-15-4-v20230907-x86-64 --image-project=opensuse-cloud
```

2. Create an image based on the disk.

```shell
gcloud compute images create hosted-providers-e2e-ci-runner-setup-x86-64-image-v1 \
 --project=<project-name> --family=hosted-providers-e2e-ci-runner-x86-64-image --source-disk=leap-15-4-x86-64 \
  --source-disk-zone=<GCP_ZONE> --labels=os=opensuse-leap,team=highlander --storage-location=us \
  --licenses https://www.googleapis.com/compute/v1/projects/vm-options/global/licenses/enable-vmx
```

3. Delete the disk.
```shell
gcloud compute disks delete leap-15-4-x86-64 --project=<project-name> --zone <GCP_ZONE>
```

4. Create an instance template using the image.
```shell
gcloud compute instance-templates create hosted-providers-e2e-ci-runner-spot-x86-64-template-v1 \
--project=<project-name> --machine-type=n2-standard-8 \
--network-interface=network=<network-name>,network-tier=PREMIUM,nic-type=VIRTIO_NET --preemptible \
--maintenance-policy=MIGRATE --provisioning-model=STANDARD \
 --service-account=<service_account> \
 --scopes=https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/trace.append \
 --create-disk=auto-delete=yes,boot=yes,device-name=hosted-providers-e2e-ci-runner-x86-64-template-v1,image=projects/<project-name>/global/images/hosted-providers-e2e-ci-runner-x86-64-image-v1,mode=rw,size=500,type=pd-ssd \
 --shielded-secure-boot --shielded-vtpm --shielded-integrity-monitoring --labels=arch=x86-64,team=hosted-providers --reservation-affinity=any
```

```shell
gcloud compute instance-templates create hosted-providers-e2e-ci-runner-spot-x86-64-template-v1 --project=<project-name> --machine-type=n2-standard-8 --network-interface=network=<network-name>,network-tier=PREMIUM --no-restart-on-failure --maintenance-policy=TERMINATE --provisioning-model=SPOT --instance-termination-action=DELETE --max-run-duration=36000s --service-account=hosted-providers-ci@<project-name>.iam.gserviceaccount.com --scopes=https://www.googleapis.com/auth/cloud-platform --create-disk=auto-delete=yes,boot=yes,device-name=hosted-providers-e2e-ci-runner-spot-x86-64-template-v1,image=projects/<project-name>/global/images/hosted-providers-e2e-ci-runner-x86-64-image-v1,mode=rw,size=30,type=pd-ssd --local-ssd=interface=NVME --no-shielded-secure-boot --shielded-vtpm --shielded-integrity-monitoring --reservation-affinity=any
```

5. Create an instance from instance template, ensure you add a local SSD with NVMe; the configuration scripts uses it for hosted runner setup.
```shell
gcloud compute instances create hosted-providers-runner-setup \
    --project=<project-name> \
    --zone=<GCP_ZONE> \
    --machine-type=n2-standard-8 \
    --network-interface=network-tier=PREMIUM,stack-type=IPV4_ONLY,subnet=hosted-providers-ci \
    --maintenance-policy=MIGRATE \
    --provisioning-model=STANDARD \
    --service-account=hosted-providers-ci@<project-name>.iam.gserviceaccount.com \
    --scopes=https://www.googleapis.com/auth/cloud-platform \
    --create-disk=auto-delete=yes,boot=yes,device-name=hosted-providers-runner-setup,image=projects/<project-name>/global/images/hosted-providers-e2e-ci-runner-setup-x86-64-image-v1,mode=rw,size=30,type=projects/<project-name>/zones/<GCP_ZONE>/diskTypes/pd-ssd \
    --local-ssd=interface=NVME \
    --no-shielded-secure-boot \
    --shielded-vtpm \
    --shielded-integrity-monitoring \
    --labels=goog-ec-src=vm_add-gcloud \
    --reservation-affinity=any
```
6. Once the instance is up, SSH into it and run the following commands:

On a virtualization capable system you need to install KVM pattern and some packages and activate the needed services, and You can check that the virtualization stack is well configured:
```shell
sudo zypper -n up -l && sudo zypper -n in -l docker jq make gcc ipxe-bootimgs && sudo zypper -n in -l -t pattern kvm_tools
sudo systemctl enable --now docker
sudo systemctl enable --now libvirtd
virt-host-validate
```

Google Cloud SDK is also needed (please adapt the version to your needs):
```shell
cd /opt
sudo curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-439.0.0-linux-x86_64.tar.gz
sudo tar xf google-cloud-cli-439.0.0-linux-x86_64.tar.gz
sudo ./google-cloud-sdk/install.sh
```

Now add the startup script /opt/configure-gh-runner.sh as root:root  with permissions set to 755. This script will configure all during VM boot to be able to add the self-hosted runner:
```shell
sudo chmod 755 /opt/configure-gh-runner.sh
```

Note: The script will only work for rancher/hosted-providers-e2e repository, to be able to use the script with other repo, change `GH_REPO` with desired repo.

To be able to automatically start this script at boot you need to create a systemd service, for this create a /etc/systemd/system/configure-gh-runner.service  file with this content.

Now activate the script during startup:
```shell
sudo systemctl daemon-reload
sudo systemctl enable configure-gh-runner
```

Create an instance template from the instance.

1. Stop the VM Instance.

2. Create an image out of the disk
```shell
gcloud compute images create hosted-providers-e2e-ci-runner-x86-64-image-v1 \
    --project=<project-name> \
    --source-disk=pvala-hosted-providers-ci-runner-setup \
    --source-disk-zone=<GCP_ZONE> \
    --storage-location=us
```

3. Create an instance template using the image with a local ssd (interface=NVMe), and spot instance (to save the cost, terminate after 10hrs). This is the instance template that will be used by the self-hosted runner.
```shell
gcloud compute instance-templates create hosted-providers-e2e-ci-runner-spot-x86-64-template-v1 \
 --project=<project-name> --machine-type=n2-standard-8 \
 --network-interface=network=<network-name>,network-tier=PREMIUM --no-restart-on-failure \
 --maintenance-policy=TERMINATE --provisioning-model=SPOT --instance-termination-action=DELETE --max-run-duration=36000s \
 --service-account=hosted-providers-ci@<project-name>.iam.gserviceaccount.com --scopes=https://www.googleapis.com/auth/cloud-platform \
 --create-disk=auto-delete=yes,boot=yes,device-name=hosted-providers-e2e-ci-runner-spot-x86-64-template-v1,image=projects/<project-name>/global/images/hosted-providers-e2e-ci-runner-setup-x86-64-image-v1,mode=rw,size=30,type=pd-ssd \
 --local-ssd=interface=NVME --no-shielded-secure-boot --shielded-vtpm --shielded-integrity-monitoring --reservation-affinity=any
```

## Secrets and Env Variables to export
### Secrets
1. `GOOGLE_APPLICATION_CREDENTIALS` - Google service account credentials with the following permissions: Compute Admin, Compute Network Admin Compute Viewer, IAP-secured Tunnel User, Kubernetes Engine Admin, Secret Manager Admin, Service Account User
2. `SELF_HOSTED_RUNNER_PAT_TOKEN` - GitHub Personal Access Token with complete permissions to actions, workflows, and pull requests.

### Environment Variables:
1. `GCP_ZONE` - Zone where the runner should be created.