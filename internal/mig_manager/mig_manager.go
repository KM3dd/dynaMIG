package mig_manager

import (
	"fmt"
	"log"
	"os"

	types "github.com/KM3dd/dynaMIG/internal/types"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

type MIGDevice types.MIGDevice

func ListMigDevices() {
	fmt.Println("MIG Device Availability Tool (NVML Version)")
	fmt.Println("==========================================")

	// Initialize NVML library
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		log.Fatalf("Failed to initialize NVML: %v", nvml.ErrorString(ret))
	}
	defer nvml.Shutdown()

	// Get driver and NVML versions
	driverVersion, ret := nvml.SystemGetDriverVersion()
	if ret != nvml.SUCCESS {
		log.Printf("Warning: Failed to get driver version: %v", nvml.ErrorString(ret))
	} else {
		fmt.Printf("Driver Version: %s\n", driverVersion)
	}

	nvmlVersion, ret := nvml.SystemGetNVMLVersion()
	if ret != nvml.SUCCESS {
		log.Printf("Warning: Failed to get NVML version: %v", nvml.ErrorString(ret))
	} else {
		fmt.Printf("NVML Version: %s\n", nvmlVersion)
	}

	// Get device count
	deviceCount, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		log.Fatalf("Failed to get device count: %v", nvml.ErrorString(ret))
	}
	fmt.Printf("Found %d GPU devices\n", deviceCount)

	// Collect all MIG devices
	var migDevices []MIGDevice

	for i := 0; i < deviceCount; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			log.Printf("Warning: Failed to get device handle for GPU %d: %v", i, nvml.ErrorString(ret))
			continue
		}

		// Get device name and check if MIG mode is enabled
		deviceName, ret := device.GetName()
		if ret != nvml.SUCCESS {
			deviceName = "Unknown"
		}

		// Check if MIG mode is enabled
		migMode, _, ret := device.GetMigMode()
		if ret != nvml.SUCCESS || migMode != nvml.DEVICE_MIG_ENABLE {
			fmt.Printf("GPU %d (%s): MIG mode not enabled\n", i, deviceName)
			continue
		}

		fmt.Printf("GPU %d (%s): MIG mode enabled\n", i, deviceName)

		// Get MIG instance count
		migInstanceCount, ret := device.GetMaxMigDeviceCount()
		if ret != nvml.SUCCESS {
			log.Printf("Warning: Failed to get max MIG device count for GPU %d: %v", i, nvml.ErrorString(ret))
			continue
		}

		// Get MIG devices on this GPU
		for j := 0; j < migInstanceCount; j++ {
			migDevice, ret := device.GetMigDeviceHandleByIndex(j)
			if ret == nvml.ERROR_NOT_FOUND {
				// This slot doesn't have a MIG device
				continue
			}
			if ret != nvml.SUCCESS {
				log.Printf("Warning: Failed to get MIG device handle for GPU %d, MIG %d: %v", i, j, nvml.ErrorString(ret))
				continue
			}

			// Get MIG device info
			migInfo := MIGDevice{
				DeviceID:   i,
				InstanceID: j,
				GPU:        deviceName,
			}

			// Get memory info
			memInfo, ret := migDevice.GetMemoryInfo()
			if ret == nvml.SUCCESS {
				migInfo.Memory = memInfo.Total / (1024 * 1024) // Convert to MB
			}

			// Check if the MIG device is in use
			// Get process information
			procInfo, ret := migDevice.GetComputeRunningProcesses()
			if ret == nvml.SUCCESS && len(procInfo) > 0 {
				migInfo.InUse = true
			}

			// Alternatively, check graphics processes
			graphicsProcInfo, ret := migDevice.GetGraphicsRunningProcesses()
			if ret == nvml.SUCCESS && len(graphicsProcInfo) > 0 {
				migInfo.InUse = true
			}

			// Get MIG profile info
			// This is a simplified approach - actual implementation would need to get profile info
			// using GetGpuInstanceProfileInfo and GetComputeInstanceProfileInfo
			gpuInstanceId, ret := migDevice.GetGpuInstanceId()
			if ret == nvml.SUCCESS {
				computeInstanceId, ret := migDevice.GetComputeInstanceId()
				if ret == nvml.SUCCESS {
					migInfo.ProfileName = fmt.Sprintf("gi-%d:ci-%d", gpuInstanceId, computeInstanceId)
				}
			}

			migDevices = append(migDevices, migInfo)
		}
	}

	// Display MIG devices
	if len(migDevices) == 0 {
		fmt.Println("No MIG devices found on this system.")
		os.Exit(0)
	}

	// Calculate counts
	var usedCount, unusedCount int
	for _, device := range migDevices {
		if device.InUse {
			usedCount++
		} else {
			unusedCount++
		}
	}

	fmt.Printf("\nFound %d MIG devices: %d used, %d available\n\n", len(migDevices), usedCount, unusedCount)

	// Print header
	fmt.Printf("%-8s %-8s %-20s %-10s %-10s %-15s\n",
		"GPU ID", "MIG ID", "GPU Name", "In Use", "Memory (MB)", "Profile")
	fmt.Println("--------------------------------------------------------------------------------")

	// Print details for each MIG device
	for _, device := range migDevices {
		inUseStr := "No"
		if device.InUse {
			inUseStr = "Yes"
		}

		fmt.Printf("%-8d %-8d %-20s %-10s %-10d %-15s\n",
			device.DeviceID, device.InstanceID, device.GPU, inUseStr, device.Memory, device.ProfileName)
	}
}

// cleanUpCiAndGi tears down the MIG compute instance and GPU instance.
func cleanUpCiAndGi(gpuid int, cid int, gid int) error {

	parent, ret := nvml.DeviceGetHandleByUUID(string(gpuid))
	if ret != nvml.SUCCESS {
		fmt.Println("error obtaining GPU handle for cleanup")
		return fmt.Errorf("unable to get device handle: %v", ret)
	}

	gi, ret := parent.GetGpuInstanceById(gid)
	if ret != nvml.SUCCESS {
		fmt.Println("error obtaining gpu instance")
		return fmt.Errorf("unable to find GI: %v", ret)
	}
	ci, ret := gi.GetComputeInstanceById(cid)
	if ret != nvml.SUCCESS {
		fmt.Println("error obtaining compute instance")
		return fmt.Errorf("unable to find CI: %v", ret)
	}
	// Destroy CI
	ret = ci.Destroy()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("unable to destroy CI: %v", ret)
	}
	// Destroy GI
	ret = gi.Destroy()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("unable to destroy GI: %v", ret)
	}

	fmt.Println("Successfully destroyed MIG resources")
	return nil

}

func CreateMigSlice(device nvml.Device, giProfileInfo nvml.GpuInstanceProfileInfo, ciProfileId int32, placement nvml.GpuInstancePlacement) error {

	fmt.Println("creating slice")
	var gi nvml.GpuInstance
	var ret nvml.Return
	gi, ret = device.CreateGpuInstanceWithPlacement(&giProfileInfo, &placement)
	if ret != nvml.SUCCESS {
		switch ret {
		case nvml.ERROR_INSUFFICIENT_RESOURCES:
			// Handle insufficient resources case
			gpuInstances, ret := device.GetGpuInstances(&giProfileInfo)
			if ret != nvml.SUCCESS {
				fmt.Println("gpu instances cannot be listed")
				return fmt.Errorf("gpu instances cannot be listed: %v", ret)
			}

			for _, gpuInstance := range gpuInstances {
				gpuInstanceInfo, ret := gpuInstance.GetInfo()
				if ret != nvml.SUCCESS {
					fmt.Println("unable to obtain gpu instance info")
					return fmt.Errorf("unable to obtain gpu instance info: %v", ret)
				}

				parentUuid, ret := gpuInstanceInfo.Device.GetUUID()
				if ret != nvml.SUCCESS {
					fmt.Println("unable to obtain parent gpu uuuid")
					return fmt.Errorf("unable to obtain parent gpu uuuid: %v", ret)
				}

				gpuUUid, ret := device.GetUUID()
				if ret != nvml.SUCCESS {
					fmt.Println("unable to obtain parent gpu uuuid")
				}
				if gpuInstanceInfo.Placement.Start == placement.Start && parentUuid == gpuUUid {
					gi, ret = device.GetGpuInstanceById(int(gpuInstanceInfo.Id))
					if ret != nvml.SUCCESS {
						fmt.Println("unable to obtain gi post iteration")
						return fmt.Errorf("unable to obtain gi post iteration, got value: %v", gi)
					}
				}
			}
		default:
			// this case is typically for scenario where ret is not equal to nvml.ERROR_INSUFFICIENT_RESOURCES
			fmt.Println(ret, "gpu instance creation errored out with unknown error")
			return fmt.Errorf("gpu instance creation failed: %v", ret)
		}
		return fmt.Errorf("error creating gpu instance profile with: %v", ret)
	}

	ciProfileInfo, ret := gi.GetComputeInstanceProfileInfo(int(ciProfileId), 0)
	if ret != nvml.SUCCESS {
		fmt.Println(ret, "error getting compute instance profile info")
		return fmt.Errorf("error getting compute instance profile info: %v", ret)
	}

	ci, ret := gi.CreateComputeInstance(&ciProfileInfo)
	if ret != nvml.SUCCESS {
		if ret != nvml.ERROR_INSUFFICIENT_RESOURCES {
			fmt.Println(ret, "error creating new compute instance, reusing", "ci", ci)
		}
	}

	return nil
}
