package k8s_manager

import (
	"fmt"
	"log"
	"os"

	types "github.com/KM3dd/dynaMIG/internal/types"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

type MIGDevice types.MIGDevice

func listMigDevices() {
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
