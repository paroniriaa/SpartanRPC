#!/bin/sh

echo "creating VM in GCP using gcloud..."

#gcloud auth login --no-browser

gcloud auth login

gcloud config set project cmpe-295-369609

gcloud compute instances create node-c --project=cmpe-295-369609 --zone=us-west1-a --machine-type=e2-small --network-interface=network-tier=PREMIUM,subnet=default --maintenance-policy=MIGRATE --provisioning-model=STANDARD --service-account=563449857226-compute@developer.gserviceaccount.com --scopes=https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/trace.append --tags=http-server,https-server --create-disk=auto-delete=yes,boot=yes,device-name=node-c,image=projects/ubuntu-os-cloud/global/images/ubuntu-1804-bionic-v20190122,mode=rw,size=10,type=projects/cmpe-295-369609/zones/us-west1-a/diskTypes/pd-balanced --reservation-affinity=any
