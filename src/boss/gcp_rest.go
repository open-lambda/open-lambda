package boss

type GcpLaunchVmArgs struct {
	ServiceAccountEmail string
	Project             string
	Region              string
	Zone                string
	InstanceName        string
	//SourceImage string
	SnapshotName string
}

type GcpSnapshotArgs struct {
	Project      string
	Region       string
	Zone         string
	Disk         string
	SnapshotName string
}

const gcpSnapshotJSON = `{
  "labels": {
    "ol-cluster": "123"
  },
  "name": "{{.SnapshotName}}",
  "sourceDisk": "projects/{{.Project}}/zones/{{.Zone}}/disks/{{.Disk}}",
  "storageLocations": [
    "{{.Region}}"
  ]
}`

const gcpLaunchVmJSON3 = `{
  "canIpForward": false,
  "confidentialInstanceConfig": {
    "enableConfidentialCompute": false
  },
  "deletionProtection": false,
  "description": "",
  "disks": [
    {
      "autoDelete": true,
      "boot": true,
      "deviceName": "{{.InstanceName}}",
      "diskEncryptionKey": {},
      "initializeParams": {
        "diskSizeGb": "10",
        "diskType": "projects/{{.Project}}/zones/{{.Zone}}/diskTypes/pd-balanced",
        "labels": {},
        "sourceImage": "{{.SourceImage}}"
      },
      "mode": "READ_WRITE",
      "type": "PERSISTENT"
    }
  ],
  "displayDevice": {
    "enableDisplay": false
  },
  "guestAccelerators": [],
  "labels": {},
  "machineType": "projects/{{.Project}}/zones/{{.Zone}}/machineTypes/e2-small",
  "metadata": {
    "items": []
  },
  "name": "{{.InstanceName}}",
  "networkInterfaces": [
    {
      "accessConfigs": [
        {
          "name": "External NAT",
          "networkTier": "PREMIUM"
        }
      ],
      "subnetwork": "projects/{{.Project}}/regions/{{.Region}}/subnetworks/default"
    }
  ],
  "reservationAffinity": {
    "consumeReservationType": "ANY_RESERVATION"
  },
  "scheduling": {
    "automaticRestart": true,
    "onHostMaintenance": "MIGRATE",
    "preemptible": false
  },
  "serviceAccounts": [
    {
      "email": "{{.ServiceAccountEmail}}",
      "scopes": [
        "https://www.googleapis.com/auth/compute",
        "https://www.googleapis.com/auth/servicecontrol",
        "https://www.googleapis.com/auth/service.management.readonly",
        "https://www.googleapis.com/auth/logging.write",
        "https://www.googleapis.com/auth/monitoring.write",
        "https://www.googleapis.com/auth/trace.append",
        "https://www.googleapis.com/auth/devstorage.read_only"
      ]
    }
  ],
  "shieldedInstanceConfig": {
    "enableIntegrityMonitoring": true,
    "enableSecureBoot": false,
    "enableVtpm": true
  },
  "tags": {
    "items": [
      "http-server",
      "https-server"
    ]
  },
  "zone": "projects/{{.Project}}/zones/{{.Zone}}"
}`

const gcpLaunchVmJSON2 = `{
{
  "canIpForward": false,
  "confidentialInstanceConfig": {
    "enableConfidentialCompute": false
  },
  "deletionProtection": false,
  "description": "",
  "disks": [
    {
      "autoDelete": true,
      "boot": true,
      "deviceName": "instance-3",
      "diskEncryptionKey": {},
      "initializeParams": {
        "diskSizeGb": "15",
        "diskType": "projects/cs320-f21/zones/us-central1-a/diskTypes/pd-balanced",
        "labels": {},
        "sourceSnapshot": "projects/cs320-f21/global/snapshots/test-snap"
      },
      "mode": "READ_WRITE",
      "type": "PERSISTENT"
    }
  ],
  "displayDevice": {
    "enableDisplay": false
  },
  "guestAccelerators": [],
  "labels": {},
  "machineType": "projects/cs320-f21/zones/us-central1-a/machineTypes/e2-small",
  "metadata": {
    "items": []
  },
  "name": "instance-3",
  "networkInterfaces": [
    {
      "accessConfigs": [
        {
          "name": "External NAT",
          "networkTier": "PREMIUM"
        }
      ],
      "subnetwork": "projects/cs320-f21/regions/us-central1/subnetworks/default"
    }
  ],
  "reservationAffinity": {
    "consumeReservationType": "ANY_RESERVATION"
  },
  "scheduling": {
    "automaticRestart": true,
    "onHostMaintenance": "MIGRATE",
    "provisioningModel": "STANDARD"
  },
  "serviceAccounts": [
    {
      "email": "1033345160415-compute@developer.gserviceaccount.com",
      "scopes": [
        "https://www.googleapis.com/auth/devstorage.read_only",
        "https://www.googleapis.com/auth/logging.write",
        "https://www.googleapis.com/auth/monitoring.write",
        "https://www.googleapis.com/auth/servicecontrol",
        "https://www.googleapis.com/auth/service.management.readonly",
        "https://www.googleapis.com/auth/trace.append"
      ]
    }
  ],
  "tags": {
    "items": []
  },
  "zone": "projects/cs320-f21/zones/us-central1-a"
}
`

const gcpLaunchVmJSON = `{
  "canIpForward": false,
  "confidentialInstanceConfig": {
    "enableConfidentialCompute": false
  },
  "deletionProtection": false,
  "description": "",
  "disks": [
    {
      "autoDelete": true,
      "boot": true,
      "deviceName": "{{.InstanceName}}",
      "diskEncryptionKey": {},
      "initializeParams": {
        "diskSizeGb": "15",
        "diskType": "projects/{{.Project}}/zones/{{.Zone}}/diskTypes/pd-balanced",
        "labels": {},
        "sourceSnapshot": "projects/{{.Project}}/global/snapshots/{{.SnapshotName}}"
      },
      "mode": "READ_WRITE",
      "type": "PERSISTENT"
    }
  ],
  "displayDevice": {
    "enableDisplay": false
  },
  "guestAccelerators": [],
  "labels": {},
  "machineType": "projects/{{.Project}}/zones/{{.Zone}}/machineTypes/e2-small",
  "metadata": {
    "items": []
  },
  "name": "{{.InstanceName}}",
  "networkInterfaces": [
    {
      "accessConfigs": [
        {
          "name": "External NAT",
          "networkTier": "PREMIUM"
        }
      ],
      "subnetwork": "projects/{{.Project}}/regions/{{.Region}}/subnetworks/default"
    }
  ],
  "reservationAffinity": {
    "consumeReservationType": "ANY_RESERVATION"
  },
  "scheduling": {
    "automaticRestart": true,
    "onHostMaintenance": "MIGRATE",
    "preemptible": false
  },
  "serviceAccounts": [
    {
      "email": "{{.ServiceAccountEmail}}",
      "scopes": [
        "https://www.googleapis.com/auth/compute",
        "https://www.googleapis.com/auth/servicecontrol",
        "https://www.googleapis.com/auth/service.management.readonly",
        "https://www.googleapis.com/auth/logging.write",
        "https://www.googleapis.com/auth/monitoring.write",
        "https://www.googleapis.com/auth/trace.append",
        "https://www.googleapis.com/auth/devstorage.read_only"
      ]
    }
  ],
  "shieldedInstanceConfig": {
    "enableIntegrityMonitoring": true,
    "enableSecureBoot": false,
    "enableVtpm": true
  },
  "tags": {
    "items": [
      "http-server",
      "https-server"
    ]
  },
  "zone": "projects/{{.Project}}/zones/{{.Zone}}"
}`
