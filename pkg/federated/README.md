# Federated Search

Use federated search to query and combine results from multiple Red Hat Advanced Cluster Management.

## Pre-requisites  
1. Multicluster Global Hub Operator 1.0.0 or later.  
2. Red Hat Advanced Cluster Management 2.10.0 or later.  
    - Managed Hub clusters mu have RHACM 2.9.0 or later

## Setup
Execute the script at `./setup.sh` to configure Global Search on the Global Hub cluster. 

```bash
bash <(curl -s https://raw.githubusercontent.com/stolostron/search-v2-api/main/pkg/federated/setup.sh) 
``` 

The script automates the following steps:
  1. Enable the Managed Service Account add-on in the MulticlusterEngine CR.
  2. Create a service account and secret to access resources managed from the Global Hub cluster.
  3. Create a route and managed service acount on each managed hub to access resources managed by each managed hub.
  4. Configure the Console to use the Global Search API.

> NOTES:
> Must run using an account with role `open-cluster-management:admin-aggregate` or higher.
> You must re-run this script when a Managed Hub is added.    
> This setup is required for Development Preview, it will be fully automated for GA.

## Uninstall
Execute the script at `./cleanup.sh` to remove the Global Search configuration from the Global Hub cluster. 

```bash
bash <(curl -s https://raw.githubusercontent.com/stolostron/search-v2-api/main/pkg/federated/cleanup.sh) 
``` 