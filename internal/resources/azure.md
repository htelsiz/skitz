# Azure

`az login` log in to Azure ^run
`az account show` show active subscription ^run
`az account list -o table` list subscriptions ^run
`az account set -s {{sub}}` set subscription ^run:sub
`az group list -o table` list resource groups ^run
`az group create -n {{name}} -l {{location}}` create resource group ^run:name
`az vm list -o table` list VMs ^run
`az vm start -g {{rg}} -n {{name}}` start VM ^run:rg
`az vm stop -g {{rg}} -n {{name}}` stop VM ^run:rg
`az vm deallocate -g {{rg}} -n {{name}}` deallocate VM ^run:rg
`az aks list -o table` list AKS clusters ^run
`az acr list -o table` list container registries ^run
`az storage account list -o table` list storage accounts ^run
`az webapp list -o table` list web apps ^run
`az functionapp list -o table` list function apps ^run
`az network vnet list -o table` list VNets ^run
`az monitor activity-log list --max-events 10` recent activity ^run
`az resource list -g {{rg}} -o table` list resources in group ^run:rg
