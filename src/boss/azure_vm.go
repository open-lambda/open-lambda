package boss

// import (
// 	"bytes"
// 	"context"
// 	"encoding/binary"
// 	"log"
// 	"net"
// 	"os"
// 	"strconv"

// 	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
// 	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
// 	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
// 	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
// 	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
// 	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
// )

// var vmName string
// var diskName string
// var newDiskName string
// var vnetName string
// var subnetName string
// var nsgName string
// var nicName string
// var publicIPName string
// var imageName string
// var snapshotName string

// const (
// 	resourceGroupName = "olvm-pool"
// 	location          = "eastus"
// )

// func iptoInt(ip string) uint32 {
// 	var long uint32
// 	ip = ip[:len(ip)-3] // xxxx.xxxx.xxxx.xxxx/24
// 	binary.Read(bytes.NewBuffer(net.ParseIP(ip).To4()), binary.BigEndian, &long)
// 	return long
// }

// func backtoIP4(ipInt int64) string {
// 	// need to do two bit shifting and “0xff” masking
// 	b0 := strconv.FormatInt((ipInt>>24)&0xff, 10)
// 	b1 := strconv.FormatInt((ipInt>>16)&0xff, 10)
// 	b2 := strconv.FormatInt((ipInt>>8)&0xff, 10)
// 	b3 := strconv.FormatInt((ipInt & 0xff), 10)
// 	b3 += "/24"
// 	return b0 + "." + b1 + "." + b2 + "." + b3
// }

// func createVM() *AzureConfig {
// 	var conf *AzureConfig
// 	if conf, err = ReadAzureConfig(); err != nil {
// 		log.Fatalf("Read to azure.json file failed\n")
// 	}
// 	//vmNum := conf.Resource_groups.Rgroup[0].Numvm

// 	conn, err := connectionAzure()
// 	if err != nil {
// 		log.Fatalf("cannot connect to Azure:%+v", err)
// 	}
// 	ctx := context.Background()

// 	log.Println("start creating virtual machine...")
// 	resourceGroup, err := createResourceGroup(ctx, conn)
// 	if err != nil {
// 		log.Fatalf("cannot create resource group:%+v", err)
// 	}
// 	log.Printf("Created resource group: %s", *resourceGroup.ID)
// 	conf.Resource_groups.Rgroup[0].Resource = *resourceGroup

// 	num_vm := conf.Resource_groups.Rgroup[0].Numvm
// 	vmName += strconv.Itoa(num_vm + 1)
// 	newDiskName = vmName + "-disk"
// 	subnetName = vmName + "-subnet"
// 	nsgName = vmName + "-nsg"
// 	nicName = vmName + "-nic"
// 	publicIPName = vmName + "-public-ip"

// 	// create snapshot
// 	disk, err := getDisk(ctx, conn)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	log.Println("Fetched disk:", *disk.ID)

// 	snapshot, err := createSnapshot(ctx, conn, *disk.ID)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	log.Println("Created snapshot:", *snapshot.ID)

// 	new_disk, err := createDisk(ctx, conn, *snapshot.ID)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	log.Println("Created disk:", *new_disk.ID)

// 	new_vm := new(vmStatus)
// 	// get network
// 	virtualNetwork, err := getVirtualNetwork(ctx, conn)
// 	if err != nil {
// 		log.Fatalf("cannot get virtual network:%+v", err)
// 	}
// 	log.Printf("Fetched virtual network: %s", *virtualNetwork.ID)
// 	new_vm.Virtual_net = *virtualNetwork

// 	// get subnets
// 	subnets := virtualNetwork.Properties.Subnets
// 	// last subnet addr
// 	lastSubnet64 := int64(iptoInt(*subnets[len(subnets)-1].Properties.AddressPrefix))
// 	lastSubnet64 += 256
// 	newSubnetIP := backtoIP4(lastSubnet64)

// 	subnet, err := createSubnets(ctx, conn, newSubnetIP)
// 	if err != nil {
// 		log.Fatalf("cannot create subnet:%+v", err)
// 	}
// 	log.Printf("Created subnet: %s", *subnet.ID)
// 	new_vm.Subnet = *subnet

// 	// publicIP, err := createPublicIP(ctx, conn)
// 	// if err != nil {
// 	// 	log.Fatalf("cannot create public IP address:%+v", err)
// 	// }
// 	// log.Printf("Created public IP address: %s", *publicIP.ID)
// 	// conf.Resource_groups.Rgroup[0].Public_ip[vmNum] = *publicIP

// 	// network security group
// 	nsg, err := createNetworkSecurityGroup(ctx, conn)
// 	if err != nil {
// 		log.Fatalf("cannot create network security group:%+v", err)
// 	}
// 	log.Printf("Created network security group: %s", *nsg.ID)
// 	new_vm.Security_group = *nsg

// 	netWorkInterface, err := createNetWorkInterfaceWithoutIp(ctx, conn, *subnet.ID, *nsg.ID)
// 	if err != nil {
// 		log.Fatalf("cannot create network interface:%+v", err)
// 	}
// 	log.Printf("Created network interface: %s", *netWorkInterface.ID)
// 	new_vm.Net_ifc = *netWorkInterface

// 	networkInterfaceID := netWorkInterface.ID

// 	// create virtual machine

// 	virtualMachine, err := createVirtualMachine(ctx, conn, *networkInterfaceID, *new_disk.ID)
// 	if err != nil {
// 		log.Fatalf("cannot create virual machine:%+v", err)
// 	}
// 	log.Printf("Created network virual machine: %s", *virtualMachine.ID)

// 	log.Println("Virtual machine created successfully")
// 	new_vm.Vm = *virtualMachine
// 	new_vm.Status = "Running"

// 	rg := &conf.Resource_groups.Rgroup[0]
// 	rg.Vms = append(rg.Vms, *new_vm)
// 	conf.Resource_groups.Numrgroup = 1
// 	conf.Resource_groups.Rgroup[0].Numvm += 1

// 	if err := WriteAzureConfig(conf); err != nil {
// 		log.Fatalf("write to azure.json file failed:%s", err)
// 	}

// 	return conf
// }

// func cleanupVM(worker *AzureWorker) {
// 	conn, err := connectionAzure()
// 	if err != nil {
// 		log.Fatalf("cannot connection Azure:%+v", err)
// 	}
// 	ctx := context.Background()

// 	log.Println("start deleting virtual machine...")
// 	err = deleteVirtualMachine(ctx, conn, worker.vmName)
// 	if err != nil {
// 		log.Fatalf("cannot delete virtual machine:%+v", err)
// 	}
// 	log.Println("deleted virtual machine")

// 	err = deleteDisk(ctx, conn, worker.diskName)
// 	if err != nil {
// 		log.Fatalf("cannot delete disk:%+v", err)
// 	}
// 	log.Println("deleted disk")

// 	err = deleteNetWorkInterface(ctx, conn, worker.nicName)
// 	if err != nil {
// 		log.Fatalf("cannot delete network interface:%+v", err)
// 	}
// 	log.Println("deleted network interface")

// 	err = deleteNetworkSecurityGroup(ctx, conn, worker.nsgName)
// 	if err != nil {
// 		log.Fatalf("cannot delete network security group:%+v", err)
// 	}
// 	log.Println("deleted network security group")

// 	if worker.publicIPName != "" {
// 		err = deletePublicIP(ctx, conn, worker.publicIPName)
// 		if err != nil {
// 			log.Fatalf("cannot delete public IP address:%+v", err)
// 		}
// 		log.Println("deleted public IP address")
// 	}

// 	err = deleteSubnets(ctx, conn, worker.vnetName, worker.subnetName)
// 	if err != nil {
// 		log.Fatalf("cannot delete subnet:%+v", err)
// 	}
// 	log.Println("deleted subnet")

// 	err = deleteVirtualNetWork(ctx, conn, worker.vnetName)
// 	if err != nil {
// 		log.Fatalf("cannot delete virtual network:%+v", err)
// 	}
// 	log.Println("deleted virtual network")

// 	// err = deleteResourceGroup(ctx, conn)
// 	// if err != nil {
// 	// 	log.Fatalf("cannot delete resource group:%+v", err)
// 	// }
// 	// log.Println("deleted resource group")
// 	log.Println("success deleted virtual machine.")
// }

// func connectionAzure() (azcore.TokenCredential, error) {
// 	cred, err := azidentity.NewDefaultAzureCredential(nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return cred, nil
// }

// func createResourceGroup(ctx context.Context, cred azcore.TokenCredential) (*armresources.ResourceGroup, error) {
// 	resourceGroupClient, err := armresources.NewResourceGroupsClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	parameters := armresources.ResourceGroup{
// 		Location: to.Ptr(location),
// 		Tags:     map[string]*string{"sample-rs-tag": to.Ptr("sample-tag")}, // resource group update tags
// 	}

// 	resp, err := resourceGroupClient.CreateOrUpdate(ctx, resourceGroupName, parameters, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &resp.ResourceGroup, nil
// }

// func deleteResourceGroup(ctx context.Context, cred azcore.TokenCredential) error {
// 	resourceGroupClient, err := armresources.NewResourceGroupsClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return err
// 	}

// 	pollerResponse, err := resourceGroupClient.BeginDelete(ctx, resourceGroupName, nil)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func getVirtualNetwork(ctx context.Context, cred azcore.TokenCredential) (*armnetwork.VirtualNetwork, error) {
// 	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := vnetClient.Get(ctx, resourceGroupName, vnetName, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &resp.VirtualNetwork, nil
// }

// func createVirtualNetwork(ctx context.Context, cred azcore.TokenCredential) (*armnetwork.VirtualNetwork, error) {
// 	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	parameters := armnetwork.VirtualNetwork{
// 		Location: to.Ptr(location),
// 		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
// 			AddressSpace: &armnetwork.AddressSpace{
// 				AddressPrefixes: []*string{
// 					to.Ptr("10.1.0.0/16"), // example 10.1.0.0/16
// 				},
// 			},
// 			Subnets: []*armnetwork.Subnet{
// 				{
// 					Name: to.Ptr(subnetName + "3"),
// 					Properties: &armnetwork.SubnetPropertiesFormat{
// 						AddressPrefix: to.Ptr("10.1.0.0/24"),
// 					},
// 				},
// 			},
// 		},
// 	}

// 	pollerResponse, err := vnetClient.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, parameters, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &resp.VirtualNetwork, nil
// }

// func deleteVirtualNetWork(ctx context.Context, cred azcore.TokenCredential, vnet string) error {
// 	vnetClient, err := armnetwork.NewVirtualNetworksClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return err
// 	}

// 	pollerResponse, err := vnetClient.BeginDelete(ctx, resourceGroupName, vnet, nil)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func createSubnets(ctx context.Context, cred azcore.TokenCredential, addr string) (*armnetwork.Subnet, error) {
// 	subnetClient, err := armnetwork.NewSubnetsClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	parameters := armnetwork.Subnet{
// 		Properties: &armnetwork.SubnetPropertiesFormat{
// 			AddressPrefix: to.Ptr(addr),
// 		},
// 	}

// 	pollerResponse, err := subnetClient.BeginCreateOrUpdate(ctx, resourceGroupName, vnetName, subnetName, parameters, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &resp.Subnet, nil
// }

// func deleteSubnets(ctx context.Context, cred azcore.TokenCredential, vnet string, subnet string) error {
// 	subnetClient, err := armnetwork.NewSubnetsClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return err
// 	}

// 	pollerResponse, err := subnetClient.BeginDelete(ctx, resourceGroupName, vnet, subnet, nil)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func createNetworkSecurityGroup(ctx context.Context, cred azcore.TokenCredential) (*armnetwork.SecurityGroup, error) {
// 	nsgClient, err := armnetwork.NewSecurityGroupsClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	parameters := armnetwork.SecurityGroup{
// 		Location: to.Ptr(location),
// 		Properties: &armnetwork.SecurityGroupPropertiesFormat{
// 			SecurityRules: []*armnetwork.SecurityRule{
// 				// Windows connection to virtual machine needs to open port 3389,RDP
// 				// inbound
// 				{
// 					Name: to.Ptr("sample_inbound_22"), //
// 					Properties: &armnetwork.SecurityRulePropertiesFormat{
// 						SourceAddressPrefix:      to.Ptr("0.0.0.0/0"),
// 						SourcePortRange:          to.Ptr("*"),
// 						DestinationAddressPrefix: to.Ptr("0.0.0.0/0"),
// 						DestinationPortRange:     to.Ptr("22"),
// 						Protocol:                 to.Ptr(armnetwork.SecurityRuleProtocolTCP),
// 						Access:                   to.Ptr(armnetwork.SecurityRuleAccessAllow),
// 						Priority:                 to.Ptr[int32](100),
// 						Description:              to.Ptr("sample network security group inbound port 22"),
// 						Direction:                to.Ptr(armnetwork.SecurityRuleDirectionInbound),
// 					},
// 				},
// 				// outbound
// 				{
// 					Name: to.Ptr("sample_outbound_22"), //
// 					Properties: &armnetwork.SecurityRulePropertiesFormat{
// 						SourceAddressPrefix:      to.Ptr("0.0.0.0/0"),
// 						SourcePortRange:          to.Ptr("*"),
// 						DestinationAddressPrefix: to.Ptr("0.0.0.0/0"),
// 						DestinationPortRange:     to.Ptr("22"),
// 						Protocol:                 to.Ptr(armnetwork.SecurityRuleProtocolTCP),
// 						Access:                   to.Ptr(armnetwork.SecurityRuleAccessAllow),
// 						Priority:                 to.Ptr[int32](100),
// 						Description:              to.Ptr("sample network security group outbound port 22"),
// 						Direction:                to.Ptr(armnetwork.SecurityRuleDirectionOutbound),
// 					},
// 				},
// 			},
// 		},
// 	}

// 	pollerResponse, err := nsgClient.BeginCreateOrUpdate(ctx, resourceGroupName, nsgName, parameters, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &resp.SecurityGroup, nil
// }

// func deleteNetworkSecurityGroup(ctx context.Context, cred azcore.TokenCredential, nsg string) error {
// 	nsgClient, err := armnetwork.NewSecurityGroupsClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return err
// 	}

// 	pollerResponse, err := nsgClient.BeginDelete(ctx, resourceGroupName, nsg, nil)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func createPublicIP(ctx context.Context, cred azcore.TokenCredential) (*armnetwork.PublicIPAddress, error) {
// 	publicIPAddressClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	parameters := armnetwork.PublicIPAddress{
// 		Location: to.Ptr(location),
// 		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
// 			PublicIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodStatic), // Static or Dynamic
// 		},
// 	}

// 	pollerResponse, err := publicIPAddressClient.BeginCreateOrUpdate(ctx, resourceGroupName, publicIPName, parameters, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &resp.PublicIPAddress, err
// }

// func deletePublicIP(ctx context.Context, cred azcore.TokenCredential, ipName string) error {
// 	publicIPAddressClient, err := armnetwork.NewPublicIPAddressesClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return err
// 	}

// 	pollerResponse, err := publicIPAddressClient.BeginDelete(ctx, resourceGroupName, ipName, nil)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func createNetWorkInterfaceWithoutIp(ctx context.Context, cred azcore.TokenCredential, subnetID string, networkSecurityGroupID string) (*armnetwork.Interface, error) {
// 	nicClient, err := armnetwork.NewInterfacesClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	parameters := armnetwork.Interface{
// 		Location: to.Ptr(location),
// 		Properties: &armnetwork.InterfacePropertiesFormat{
// 			//NetworkSecurityGroup:
// 			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
// 				{
// 					Name: to.Ptr("ipConfig"),
// 					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
// 						PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodDynamic),
// 						Subnet: &armnetwork.Subnet{
// 							ID: to.Ptr(subnetID),
// 						},
// 					},
// 				},
// 			},
// 			NetworkSecurityGroup: &armnetwork.SecurityGroup{
// 				ID: to.Ptr(networkSecurityGroupID),
// 			},
// 		},
// 	}

// 	pollerResponse, err := nicClient.BeginCreateOrUpdate(ctx, resourceGroupName, nicName, parameters, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &resp.Interface, err
// }

// func createNetWorkInterface(ctx context.Context, cred azcore.TokenCredential, subnetID string, publicIPID string, networkSecurityGroupID string) (*armnetwork.Interface, error) {
// 	nicClient, err := armnetwork.NewInterfacesClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	parameters := armnetwork.Interface{
// 		Location: to.Ptr(location),
// 		Properties: &armnetwork.InterfacePropertiesFormat{
// 			//NetworkSecurityGroup:
// 			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
// 				{
// 					Name: to.Ptr("ipConfig"),
// 					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
// 						PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodDynamic),
// 						Subnet: &armnetwork.Subnet{
// 							ID: to.Ptr(subnetID),
// 						},
// 						PublicIPAddress: &armnetwork.PublicIPAddress{
// 							ID: to.Ptr(publicIPID),
// 						},
// 					},
// 				},
// 			},
// 			NetworkSecurityGroup: &armnetwork.SecurityGroup{
// 				ID: to.Ptr(networkSecurityGroupID),
// 			},
// 		},
// 	}

// 	pollerResponse, err := nicClient.BeginCreateOrUpdate(ctx, resourceGroupName, nicName, parameters, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &resp.Interface, err
// }

// func deleteNetWorkInterface(ctx context.Context, cred azcore.TokenCredential, nic string) error {
// 	nicClient, err := armnetwork.NewInterfacesClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return err
// 	}

// 	pollerResponse, err := nicClient.BeginDelete(ctx, resourceGroupName, nic, nil)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func createVirtualMachine(ctx context.Context, cred azcore.TokenCredential, networkInterfaceID string, new_diskID string) (*armcompute.VirtualMachine, error) {
// 	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	//require ssh key for authentication on linux
// 	//sshPublicKeyPath := "/home/user/.ssh/id_rsa.pub"
// 	//var sshBytes []byte
// 	//_,err := os.Stat(sshPublicKeyPath)
// 	//if err == nil {
// 	//	sshBytes,err = ioutil.ReadFile(sshPublicKeyPath)
// 	//	if err != nil {
// 	//		return nil, err
// 	//	}
// 	//}

// 	parameters := armcompute.VirtualMachine{
// 		Location: to.Ptr(location),
// 		Identity: &armcompute.VirtualMachineIdentity{
// 			Type: to.Ptr(armcompute.ResourceIdentityTypeNone),
// 		},
// 		Properties: &armcompute.VirtualMachineProperties{
// 			StorageProfile: &armcompute.StorageProfile{
// 				// ImageReference: &armcompute.ImageReference{
// 				// 	// search image reference
// 				// 	// az vm image list --output table
// 				// 	// Offer:     to.Ptr("WindowsServer"),
// 				// 	// Publisher: to.Ptr("MicrosoftWindowsServer"),
// 				// 	// SKU:       to.Ptr("2019-Datacenter"),
// 				// 	// Version:   to.Ptr("latest"),
// 				// 	//require ssh key for authentication on linux
// 				// 	Offer:     to.Ptr("UbuntuServer"),
// 				// 	Publisher: to.Ptr("Canonical"),
// 				// 	SKU:       to.Ptr("18.04-LTS"),
// 				// 	Version:   to.Ptr("latest"),
// 				// },
// 				OSDisk: &armcompute.OSDisk{
// 					Name:         to.Ptr(newDiskName),
// 					CreateOption: to.Ptr(armcompute.DiskCreateOptionTypesAttach),
// 					Caching:      to.Ptr(armcompute.CachingTypesReadWrite),
// 					ManagedDisk: &armcompute.ManagedDiskParameters{
// 						StorageAccountType: to.Ptr(armcompute.StorageAccountTypesStandardSSDLRS), // OSDisk type Standard/Premium HDD/SSD
// 						ID:                 to.Ptr(new_diskID),
// 					},
// 					OSType: to.Ptr(armcompute.OperatingSystemTypesLinux),
// 					//DiskSizeGB: to.Ptr[int32](100), // default 127G
// 				},
// 			},
// 			HardwareProfile: &armcompute.HardwareProfile{
// 				// TODO: make it user's choice
// 				VMSize: to.Ptr(armcompute.VirtualMachineSizeTypes("Standard_B1ms")), // VM size include vCPUs,RAM,Data Disks,Temp storage.
// 			},
// 			// OSProfile: &armcompute.OSProfile{ //
// 			// 	ComputerName:  to.Ptr(vmName),
// 			// 	AdminUsername: to.Ptr("ol-user"),
// 			// 	AdminPassword: to.Ptr("123456"),
// 			// 	//require ssh key for authentication on linux
// 			// 	//LinuxConfiguration: &armcompute.LinuxConfiguration{
// 			// 	//	DisablePasswordAuthentication: to.Ptr(true),
// 			// 	//	SSH: &armcompute.SSHConfiguration{
// 			// 	//		PublicKeys: []*armcompute.SSHPublicKey{
// 			// 	//			{
// 			// 	//				Path:    to.Ptr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", "sample-user")),
// 			// 	//				KeyData: to.Ptr(string(sshBytes)),
// 			// 	//			},
// 			// 	//		},
// 			// 	//	},
// 			// 	//},
// 			// },
// 			NetworkProfile: &armcompute.NetworkProfile{
// 				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
// 					{
// 						ID: to.Ptr(networkInterfaceID),
// 					},
// 				},
// 			},
// 		},
// 	}

// 	pollerResponse, err := vmClient.BeginCreateOrUpdate(ctx, resourceGroupName, vmName, parameters, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &resp.VirtualMachine, nil
// }

// func deleteVirtualMachine(ctx context.Context, cred azcore.TokenCredential, name string) error {
// 	vmClient, err := armcompute.NewVirtualMachinesClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return err
// 	}

// 	pollerResponse, err := vmClient.BeginDelete(ctx, resourceGroupName, name, nil)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func createSnapshotImage() {
// 	subscriptionId = os.Getenv("AZURE_SUBSCRIPTION_ID")
// 	if len(subscriptionId) == 0 {
// 		log.Fatal("AZURE_SUBSCRIPTION_ID is not set.")
// 	}

// 	TenantID := os.Getenv("AZURE_TENANT_ID")
// 	if len(TenantID) == 0 {
// 		log.Fatal("AZURE_TENANT_ID is not set.")
// 	}

// 	cred, err := azidentity.NewDefaultAzureCredential(nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	ctx := context.Background()

// 	resourceGroup, err := createResourceGroup(ctx, cred)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	log.Println("resources group:", *resourceGroup.ID)

// 	disk, err := getDisk(ctx, cred)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	snapshot, err := createSnapshot(ctx, cred, *disk.ID)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	log.Println("snapshot:", *snapshot.ID)
// }

// func getDisk(ctx context.Context, cred azcore.TokenCredential) (*armcompute.Disk, error) {
// 	diskClient, err := armcompute.NewDisksClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := diskClient.Get(
// 		ctx,
// 		resourceGroupName,
// 		diskName,
// 		nil,
// 	)

// 	if err != nil {
// 		return nil, err
// 	}

// 	return &resp.Disk, nil
// }

// func createDisk(ctx context.Context, cred azcore.TokenCredential, source_disk string) (*armcompute.Disk, error) {
// 	disksClient, err := armcompute.NewDisksClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	pollerResp, err := disksClient.BeginCreateOrUpdate(
// 		ctx,
// 		resourceGroupName,
// 		newDiskName,
// 		armcompute.Disk{
// 			Location: to.Ptr(location),
// 			SKU: &armcompute.DiskSKU{
// 				Name: to.Ptr(armcompute.DiskStorageAccountTypesStandardSSDLRS),
// 			},
// 			Properties: &armcompute.DiskProperties{
// 				CreationData: &armcompute.CreationData{
// 					CreateOption:     to.Ptr(armcompute.DiskCreateOptionCopy),
// 					SourceResourceID: to.Ptr(source_disk),
// 				},
// 				DiskSizeGB: to.Ptr[int32](64),
// 			},
// 		},
// 		nil,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := pollerResp.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &resp.Disk, nil
// }

// func deleteDisk(ctx context.Context, cred azcore.TokenCredential, disk string) error {
// 	diskClient, err := armcompute.NewDisksClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return err
// 	}

// 	pollerResponse, err := diskClient.BeginDelete(ctx, resourceGroupName, disk, nil)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = pollerResponse.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func createSnapshot(ctx context.Context, cred azcore.TokenCredential, diskID string) (*armcompute.Snapshot, error) {
// 	snapshotClient, err := armcompute.NewSnapshotsClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	pollerResp, err := snapshotClient.BeginCreateOrUpdate(
// 		ctx,
// 		resourceGroupName,
// 		snapshotName,
// 		armcompute.Snapshot{
// 			Location: to.Ptr(location),
// 			Properties: &armcompute.SnapshotProperties{
// 				CreationData: &armcompute.CreationData{
// 					CreateOption:     to.Ptr(armcompute.DiskCreateOptionCopy),
// 					SourceResourceID: to.Ptr(diskID),
// 				},
// 			},
// 		},
// 		nil,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := pollerResp.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &resp.Snapshot, nil
// }

// func cleanupSnapshot(ctx context.Context, cred azcore.TokenCredential) error {
// 	resourceGroupClient, err := armresources.NewResourceGroupsClient(subscriptionId, cred, nil)
// 	if err != nil {
// 		return err
// 	}

// 	pollerResp, err := resourceGroupClient.BeginDelete(ctx, resourceGroupName, nil)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = pollerResp.PollUntilDone(ctx, nil)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
