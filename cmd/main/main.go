package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/KM3dd/dynaMIG/internal/config"
	"github.com/KM3dd/dynaMIG/internal/mig_manager"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

func main() {
	// usage :
	// 1. Show status : GPUs lists with slices and indexes
	// 2. list usable slices : - GPU i : indexes
	// 3. create -g [GPU_ID] -p [profile-name] -s [start index]
	// 4. delete -g [GPU_ID] -p [profile-name] -s [start index]
	action := os.Args[1]
	fmt.Println("This is dynamic mig manager !")

	switch action {
	case "create":
		gpuid, _ := strconv.Atoi(os.Args[2])
		profile := os.Args[3]
		start, _ := strconv.Atoi(os.Args[4])
		create(gpuid, profile, uint32(start))
	case "list":
		mig_manager.ListMigDevices()
	}

}

func create(gpuid int, profile string, start uint32) {

	// 1. get device object
	device, retCode := nvml.DeviceGetHandleByIndex(gpuid)
	if retCode != nvml.SUCCESS {
		fmt.Println(retCode, "error getting GPU device handle")
		os.Exit(1) // Exit on error
	}

	// 2. get gi info
	giProfileID := config.A30_PROFILES[profile].GID
	ciProfileID := config.A30_PROFILES[profile].CID
	giProfileInfo, retGI := device.GetGpuInstanceProfileInfo(giProfileID)
	if retGI != nvml.SUCCESS {
		fmt.Printf("Failed to get GPU instance profile info: %v\n", retGI)
		os.Exit(1) // Exit
	}

	fmt.Println("The profile info === %v", giProfileInfo)
	// 3. make placement object
	size := config.A30_PROFILES[profile].Size
	placement := nvml.GpuInstancePlacement{
		Start: start,
		Size:  size,
	}

	// 4. create the mig with mig manager
	mig_manager.CreateMigSlice(device, giProfileInfo, ciProfileID, placement)
}
