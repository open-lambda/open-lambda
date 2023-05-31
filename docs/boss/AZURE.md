In order for an Azure Blob to be created, an existing Azure storage account
needs to be used in the url in line 115 of azure.go:

url := "https://<StorageAccountName>.blob.core.windows.net/" //replace <StorageAccountName> with your Azure storage account name

The IAM roles of storage blob data contributor and storage queue data contributor 
are both necessary to add to your Azure storage account to be able to upload and download
to the blob.

You can run Azure blob commands by running "./ol azure-test"