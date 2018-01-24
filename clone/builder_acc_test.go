package clone

import (
	builderT "github.com/hashicorp/packer/helper/builder/testing"
	commonT "github.com/jetbrains-infra/packer-builder-vsphere/common/testing"

	"github.com/hashicorp/packer/packer"
	"github.com/jetbrains-infra/packer-builder-vsphere/driver"
	"testing"
	"github.com/jetbrains-infra/packer-builder-vsphere/common"
)

func TestBuilderAcc_default(t *testing.T) {
	config := defaultConfig()
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: commonT.RenderConfig(config),
		Check:    checkDefault(t, config["vm_name"].(string), config["host"].(string), "datastore1"),
	})
}

func defaultConfig() map[string]interface{} {
	config := map[string]interface{}{
		"vcenter_server":      "vcenter.vsphere65.test",
		"username":            "root",
		"password":            "jetbrains",
		"insecure_connection": true,

		"template": "alpine",
		"host":     "esxi-1.vsphere65.test",

		"ssh_username": "root",
		"ssh_password": "jetbrains",
	}
	config["vm_name"] = commonT.NewVMName()
	return config
}

func checkDefault(t *testing.T, name string, host string, datastore string) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		d := testConn(t)
		vm := getVM(t, d, artifacts)

		vmInfo, err := vm.Info("name", "parent", "runtime.host", "resourcePool", "datastore", "layoutEx.disk")
		if err != nil {
			t.Fatalf("Cannot read VM properties: %v", err)
		}

		if vmInfo.Name != name {
			t.Errorf("Invalid VM name: expected '%v', got '%v'", name, vmInfo.Name)
		}

		f := d.NewFolder(vmInfo.Parent)
		folderPath, err := f.Path()
		if err != nil {
			t.Fatalf("Cannot read folder name: %v", err)
		}
		if folderPath != "" {
			t.Errorf("Invalid folder: expected '/', got '%v'", folderPath)
		}

		h := d.NewHost(vmInfo.Runtime.Host)
		hostInfo, err := h.Info("name")
		if err != nil {
			t.Fatal("Cannot read host properties: ", err)
		}
		if hostInfo.Name != host {
			t.Errorf("Invalid host name: expected '%v', got '%v'", host, hostInfo.Name)
		}

		p := d.NewResourcePool(vmInfo.ResourcePool)
		poolPath, err := p.Path()
		if err != nil {
			t.Fatalf("Cannot read resource pool name: %v", err)
		}
		if poolPath != "" {
			t.Errorf("Invalid resource pool: expected '/', got '%v'", poolPath)
		}

		dsr := vmInfo.Datastore[0].Reference()
		ds := d.NewDatastore(&dsr)
		dsInfo, err := ds.Info("name")
		if err != nil {
			t.Fatal("Cannot read datastore properties: ", err)
		}
		if dsInfo.Name != datastore {
			t.Errorf("Invalid datastore name: expected '%v', got '%v'", datastore, dsInfo.Name)
		}

		if len(vmInfo.LayoutEx.Disk[0].Chain) != 1 {
			t.Error("Not a full clone")
		}

		return nil
	}
}

func TestBuilderAcc_artifact(t *testing.T) {
	config := defaultConfig()
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: commonT.RenderConfig(config),
		Check:    checkArtifact(t),
	})
}

func checkArtifact(t *testing.T) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		if len(artifacts) > 1 {
			t.Fatal("more than 1 artifact")
		}

		artifactRaw := artifacts[0]
		_, ok := artifactRaw.(*common.Artifact)
		if !ok {
			t.Fatalf("unknown artifact: %#v", artifactRaw)
		}

		return nil
	}
}

func TestBuilderAcc_folder(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: folderConfig(),
		Check:    checkFolder(t, "folder1/folder2"),
	})
}

func folderConfig() string {
	config := defaultConfig()
	config["folder"] = "folder1/folder2"
	config["linked_clone"] = true // speed up
	return commonT.RenderConfig(config)
}

func checkFolder(t *testing.T, folder string) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		d := testConn(t)
		vm := getVM(t, d, artifacts)

		vmInfo, err := vm.Info("parent")
		if err != nil {
			t.Fatalf("Cannot read VM properties: %v", err)
		}

		f := d.NewFolder(vmInfo.Parent)
		path, err := f.Path()
		if err != nil {
			t.Fatalf("Cannot read folder name: %v", err)
		}
		if path != folder {
			t.Errorf("Wrong folder. expected: %v, got: %v", folder, path)
		}

		return nil
	}
}

func TestBuilderAcc_resourcePool(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: resourcePoolConfig(),
		Check:    checkResourcePool(t, "pool1/pool2"),
	})
}

func resourcePoolConfig() string {
	config := defaultConfig()
	config["resource_pool"] = "pool1/pool2"
	config["linked_clone"] = true // speed up
	return commonT.RenderConfig(config)
}

func checkResourcePool(t *testing.T, pool string) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		d := testConn(t)
		vm := getVM(t, d, artifacts)

		vmInfo, err := vm.Info("resourcePool")
		if err != nil {
			t.Fatalf("Cannot read VM properties: %v", err)
		}

		p := d.NewResourcePool(vmInfo.ResourcePool)
		path, err := p.Path()
		if err != nil {
			t.Fatalf("Cannot read resource pool name: %v", err)
		}
		if path != pool {
			t.Errorf("Wrong folder. expected: %v, got: %v", pool, path)
		}

		return nil
	}
}

func TestBuilderAcc_datastore(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: datastoreConfig(),
		Check:    checkDatastore(t, "datastore1"), // on esxi-1.vsphere65.test
	})
}

func datastoreConfig() string {
	config := defaultConfig()
	config["template"] = "alpine-host4" // on esxi-4.vsphere65.test
	return commonT.RenderConfig(config)
}

func checkDatastore(t *testing.T, name string) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		d := testConn(t)
		vm := getVM(t, d, artifacts)

		vmInfo, err := vm.Info("datastore")
		if err != nil {
			t.Fatalf("Cannot read VM properties: %v", err)
		}

		n := len(vmInfo.Datastore)
		if n != 1 {
			t.Fatalf("VM should have 1 datastore, got %v", n)
		}

		ds := d.NewDatastore(&vmInfo.Datastore[0])
		info, err := ds.Info("name")
		if err != nil {
			t.Fatalf("Cannot read datastore properties: %v", err)
		}
		if info.Name != name {
			t.Errorf("Wrong datastore. expected: %v, got: %v", name, info.Name)
		}

		return nil
	}
}

func TestBuilderAcc_multipleDatastores(t *testing.T) {
	t.Skip("test must fail")

	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: multipleDatastoresConfig(),
	})
}

func multipleDatastoresConfig() string {
	config := defaultConfig()
	config["host"] = "esxi-4.vsphere65.test" // host with 2 datastores
	return commonT.RenderConfig(config)
}

func TestBuilderAcc_linkedClone(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: linkedCloneConfig(),
		Check:    checkLinkedClone(t),
	})
}

func linkedCloneConfig() string {
	config := defaultConfig()
	config["linked_clone"] = true
	return commonT.RenderConfig(config)
}

func checkLinkedClone(t *testing.T) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		d := testConn(t)
		vm := getVM(t, d, artifacts)

		vmInfo, err := vm.Info("layoutEx.disk")
		if err != nil {
			t.Fatalf("Cannot read VM properties: %v", err)
		}

		if len(vmInfo.LayoutEx.Disk[0].Chain) != 2 {
			t.Error("Not a linked clone")
		}

		return nil
	}
}

func TestBuilderAcc_hardware(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: hardwareConfig(),
		Check:    checkHardware(t),
	})
}

func hardwareConfig() string {
	config := defaultConfig()
	config["CPUs"] = 2
	config["CPU_reservation"] = 1000
	config["CPU_limit"] = 1500
	config["RAM"] = 2048
	config["RAM_reservation"] = 1024
	config["linked_clone"] = true // speed up

	return commonT.RenderConfig(config)
}

func checkHardware(t *testing.T) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		d := testConn(t)

		vm := getVM(t, d, artifacts)
		vmInfo, err := vm.Info("config")
		if err != nil {
			t.Fatalf("Cannot read VM properties: %v", err)
		}

		cpuSockets := vmInfo.Config.Hardware.NumCPU
		if cpuSockets != 2 {
			t.Errorf("VM should have 2 CPU sockets, got %v", cpuSockets)
		}

		cpuReservation := vmInfo.Config.CpuAllocation.GetResourceAllocationInfo().Reservation
		if cpuReservation != 1000 {
			t.Errorf("VM should have CPU reservation for 1000 Mhz, got %v", cpuReservation)
		}

		cpuLimit := vmInfo.Config.CpuAllocation.GetResourceAllocationInfo().Limit
		if cpuLimit != 1500 {
			t.Errorf("VM should have CPU reservation for 1500 Mhz, got %v", cpuLimit)
		}

		ram := vmInfo.Config.Hardware.MemoryMB
		if ram != 2048 {
			t.Errorf("VM should have 2048 MB of RAM, got %v", ram)
		}

		ramReservation := vmInfo.Config.MemoryAllocation.GetResourceAllocationInfo().Reservation
		if ramReservation != 1024 {
			t.Errorf("VM should have RAM reservation for 1024 MB, got %v", ramReservation)
		}

		return nil
	}
}

func TestBuilderAcc_RAMReservation(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: RAMReservationConfig(),
		Check:    checkRAMReservation(t),
	})
}

func RAMReservationConfig() string {
	config := defaultConfig()
	config["RAM_reserve_all"] = true
	config["linked_clone"] = true // speed up

	return commonT.RenderConfig(config)
}

func checkRAMReservation(t *testing.T) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		d := testConn(t)

		vm := getVM(t, d, artifacts)
		vmInfo, err := vm.Info("config")
		if err != nil {
			t.Fatalf("Cannot read VM properties: %v", err)
		}

		if *vmInfo.Config.MemoryReservationLockedToMax != true {
			t.Errorf("VM should have all RAM reserved")
		}

		return nil
	}
}

func TestBuilderAcc_sshKey(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: sshKeyConfig(),
	})
}

func sshKeyConfig() string {
	config := defaultConfig()
	config["ssh_password"] = ""
	config["ssh_private_key_file"] = "../test-key.pem"
	config["linked_clone"] = true // speed up
	return commonT.RenderConfig(config)
}

func TestBuilderAcc_snapshot(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: snapshotConfig(),
		Check:    checkSnapshot(t),
	})
}

func snapshotConfig() string {
	config := defaultConfig()
	config["create_snapshot"] = true
	return commonT.RenderConfig(config)
}

func checkSnapshot(t *testing.T) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		d := testConn(t)

		vm := getVM(t, d, artifacts)
		vmInfo, err := vm.Info("layoutEx.disk")
		if err != nil {
			t.Fatalf("Cannot read VM properties: %v", err)
		}

		layers := len(vmInfo.LayoutEx.Disk[0].Chain)
		if layers != 2 {
			t.Errorf("VM should have a single snapshot. expected 2 disk layers, got %v", layers)
		}

		return nil
	}
}

func TestBuilderAcc_template(t *testing.T) {
	builderT.Test(t, builderT.TestCase{
		Builder:  &Builder{},
		Template: templateConfig(),
		Check:    checkTemplate(t),
	})
}

func templateConfig() string {
	config := defaultConfig()
	config["convert_to_template"] = true
	config["linked_clone"] = true // speed up
	return commonT.RenderConfig(config)
}

func checkTemplate(t *testing.T) builderT.TestCheckFunc {
	return func(artifacts []packer.Artifact) error {
		d := testConn(t)

		vm := getVM(t, d, artifacts)
		vmInfo, err := vm.Info("config.template")
		if err != nil {
			t.Fatalf("Cannot read VM properties: %v", err)
		}

		if vmInfo.Config.Template != true {
			t.Error("Not a template")
		}

		return nil
	}
}

func testConn(t *testing.T) *driver.Driver {
	d, err := driver.NewDriver(&driver.ConnectConfig{
		VCenterServer:      "vcenter.vsphere65.test",
		Username:           "root",
		Password:           "jetbrains",
		InsecureConnection: true,
	})
	if err != nil {
		t.Fatal("Cannot connect: ", err)
	}
	return d
}

func getVM(t *testing.T, d *driver.Driver, artifacts []packer.Artifact) *driver.VirtualMachine {
	artifactRaw := artifacts[0]
	artifact, _ := artifactRaw.(*common.Artifact)

	vm, err := d.FindVM(artifact.Name)
	if err != nil {
		t.Fatalf("Cannot find VM: %v", err)
	}

	return vm
}