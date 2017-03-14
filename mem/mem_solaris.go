package mem

import (
	"errors"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/internal/common"
)

// VirtualMemory for Solaris is a minimal implementation which only returns
// what Nomad needs. It does take into account global vs zone, however.
func VirtualMemory() (*VirtualMemoryStat, error) {
	result := &VirtualMemoryStat{}

	zoneName, err := zoneName()
	if err != nil {
		return nil, err
	}

	if zoneName == "global" {
		cap, err := globalZoneMemoryCapacity()
		if err != nil {
			return nil, err
		}
		result.Total = cap
	} else {
		cap, err := nonGlobalZoneMemoryCapacity(zoneName)
		if err != nil {
			return nil, err
		}
		result.Total = cap
	}

	return result, nil
}

func SwapMemory() (*SwapMemoryStat, error) {
	return nil, common.ErrNotImplementedError
}

func zoneName() (string, error) {
	zonename, err := exec.LookPath("/usr/bin/zonename")
	if err != nil {
		return "", err
	}

	out, err := invoke.Command(zonename)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

var globalZoneMemoryCapacityMatch = regexp.MustCompile(`Memory size: ([\d]+) Megabytes`)

func globalZoneMemoryCapacity() (uint64, error) {
	prtconf, err := exec.LookPath("/usr/sbin/prtconf")
	if err != nil {
		return 0, err
	}

	out, err := invoke.Command(prtconf)
	if err != nil {
		return 0, err
	}

	match := globalZoneMemoryCapacityMatch.FindAllStringSubmatch(string(out), -1)
	if len(match) != 1 {
		return 0, errors.New("Memory size not contained in output of /usr/sbin/prtconf")
	}

	return strconv.ParseUint(match[0][1], 10, 64)
}

func nonGlobalZoneMemoryCapacity(zoneName string) (uint64, error) {
	zonememstat, err := exec.LookPath("/usr/bin/zonememstat")
	if err != nil {
		return 0, err
	}

	out, err := invoke.Command(zonememstat, "-H", "-z", zoneName)
	if err != nil {
		return 0, err
	}

	const capOffset = 2
	fields := strings.Fields(string(out))

	// Zone ID, RSS, **CAP**, NOVER, POUT, SWAP%
	if len(fields) < 3 || fields[capOffset] == "" {
		return 0, errors.New("Cannot find memory capacity for non-global zone")
	}

	return strconv.ParseUint(fields[capOffset], 10, 64)
}
