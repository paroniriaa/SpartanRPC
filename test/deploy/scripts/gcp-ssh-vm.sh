#!/bin/sh

while getopts e: flag; do
    case "${flag}" in
    e) EXTERNAL_IP=${OPTARG} ;;
    esac
done

echo "Connecting to VM $USER@$EXTERNAL_IP in GCP using ssh..."

ssh -i ~/.ssh/google_cloud_key $USER@$EXTERNAL_IP
