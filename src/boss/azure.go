package boss

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

var url string
var data []byte
var ctx context.Context
var blobClient azblob.BlockBlobClient
var blobName string
var containerName string
var err error
var containerClient azblob.ContainerClient
var subscriptionId string

func Create(contents string) {
	url := "https://openlambda.blob.core.windows.net/" //replace <StorageAccountName> with your Azure storage account name
	ctx := context.Background()
	// Create a default request pipeline using your storage account name and account key.
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
	}
	serviceClient, err := azblob.NewServiceClient(url, credential, nil)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
	}
	containerName := fmt.Sprintf("quickstart-%s", randomString())
	fmt.Printf("Creating a container named %s\n", containerName)
	containerClient, err := serviceClient.NewContainerClient(containerName)
	if err != nil {
		log.Fatal(err)
	}
	_, err = containerClient.Create(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Creating a dummy file to test the upload and download\n")

	data := []byte(contents)
	blobName := "quickstartblob" + "-" + randomString()

	blobClient, err := azblob.NewBlockBlobClient(url+containerName+"/"+blobName, credential, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Upload to data to blob storage
	_, err = blobClient.UploadBuffer(ctx, data, azblob.UploadOption{})

	if err != nil {
		log.Fatalf("Failure to upload to blob: %+v", err)
	}
}

func Download() {
	// Download the blob
	get, err := blobClient.Download(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	downloadedData := &bytes.Buffer{}
	reader := get.Body(&azblob.RetryReaderOptions{})
	_, err = downloadedData.ReadFrom(reader)
	if err != nil {
		log.Fatal(err)
	}
	err = reader.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(downloadedData.String())

	fmt.Printf("Press enter key to delete the blob fils, example container, and exit the application.\n")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	fmt.Printf("Cleaning up.\n")
}

func Delete() {
	// Delete the blob
	fmt.Printf("Deleting the blob " + blobName + "\n")

	_, err = blobClient.Delete(ctx, nil)
	if err != nil {
		log.Fatalf("Failure: %+v", err)
	}

	// Delete the container
	fmt.Printf("Deleting the blob " + containerName + "\n")
	_, err = containerClient.Delete(ctx, nil)

	if err != nil {
		log.Fatalf("Failure: %+v", err)
	}
}

func randomString() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return strconv.Itoa(r.Int())
}

func AzureCreateVM() *AzureConfig {
	subscriptionId = os.Getenv("AZURE_SUBSCRIPTION_ID")
	if len(subscriptionId) == 0 {
		log.Fatal("AZURE_SUBSCRIPTION_ID is not set.")
	}

	// TODO: generate vmName and subnet Addr automatically
	vmName = "ol-worker-"
	diskName = "ol-boss_OsDisk_1_58ab03cfbf114ad58532c893535a70ec"
	newDiskName = vmName + "-disk"
	vnetName = "ol-boss-vnet"
	subnetName = vmName + "-subnet"
	nsgName = vmName + "-nsg"
	nicName = vmName + "-nic"
	publicIPName = vmName + "-public-ip"
	imageName = "ol-boss-img"
	snapshotName = "ol-boss-snapshot"
	return createVM()
	// TODO: copy the snapshot to the new VM
	//createSnapshotImage()
}

func AzureMain(contents string) {
	fmt.Printf("Azure Blob storage quick start sample\n")

	url := "https://<StorageAccountName>.blob.core.windows.net/" //replace <StorageAccountName> with your Azure storage account name
	ctx := context.Background()

	// Create a default request pipeline using your storage account name and account key.
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
	}

	serviceClient, err := azblob.NewServiceClient(url, credential, nil)
	if err != nil {
		log.Fatal("Invalid credentials with error: " + err.Error())
	}

	// Create the container
	containerName := fmt.Sprintf("quickstart-%s", randomString())
	fmt.Printf("Creating a container named %s\n", containerName)
	containerClient, err := serviceClient.NewContainerClient(containerName)
	if err != nil {
		log.Fatal(err)
	}
	_, err = containerClient.Create(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Creating a dummy file to test the upload and download\n")

	data := []byte(contents)
	blobName := "quickstartblob" + "-" + randomString()

	blobClient, err := azblob.NewBlockBlobClient(url+containerName+"/"+blobName, credential, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Upload to data to blob storage
	_, err = blobClient.UploadBuffer(ctx, data, azblob.UploadOption{})

	if err != nil {
		log.Fatalf("Failure to upload to blob: %+v", err)
	}

	// List the blobs in the container
	fmt.Println("Listing the blobs in the container:")

	pager := containerClient.ListBlobsFlat(nil)

	for pager.NextPage(ctx) {
		resp := pager.PageResponse()

		for _, v := range resp.Segment.BlobItems {
			fmt.Println(*v.Name)
		}
	}

	if err = pager.Err(); err != nil {
		log.Fatalf("Failure to list blobs: %+v", err)
	}

	// Download the blob
	get, err := blobClient.Download(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	downloadedData := &bytes.Buffer{}
	reader := get.Body(&azblob.RetryReaderOptions{})
	_, err = downloadedData.ReadFrom(reader)
	if err != nil {
		log.Fatal(err)
	}
	err = reader.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(downloadedData.String())

	fmt.Printf("Press enter key to delete the blob fils, example container, and exit the application.\n")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	fmt.Printf("Cleaning up.\n")

	// Delete the blob
	fmt.Printf("Deleting the blob " + blobName + "\n")

	_, err = blobClient.Delete(ctx, nil)
	if err != nil {
		log.Fatalf("Failure: %+v", err)
	}

	// Delete the container
	fmt.Printf("Deleting the blob " + containerName + "\n")
	_, err = containerClient.Delete(ctx, nil)

	if err != nil {
		log.Fatalf("Failure: %+v", err)
	}
}
