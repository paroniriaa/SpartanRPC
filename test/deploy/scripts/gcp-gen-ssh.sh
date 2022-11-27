#!/bin/sh

echo "Generating default GCP usage SSH keys using ssh-keygen..."

ssh-keygen -t rsa -f ~/.ssh/google_cloud_key -C $USER

cat /home/$USER/.ssh/google_cloud_key.pub
