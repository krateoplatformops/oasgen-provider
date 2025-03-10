package versionsort

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func ptr[V any](v V) *V {
	return &v
}

func deptr[V any](v *V) V {
	if v == nil {
		return *new(V)
	}
	return *v
}

type Version struct {
	Major      *int
	Stability  *string // GA, Beta, Alpha
	Additional *int

	OriginalStr string
}

func (v *Version) String() string {
	if v == nil {
		return ""
	}

	return v.OriginalStr
}

func NewVersion(versionString string) *Version {
	v := &Version{}
	v.OriginalStr = versionString
	parts := strings.Split(versionString, "v")
	if len(parts) > 1 {
		vString := parts[1]
		var err error
		maj, err := strconv.Atoi(strings.SplitN(vString, "beta", 2)[0])
		v.Major = ptr(maj)
		if err != nil {
			maj, err := strconv.Atoi(strings.SplitN(vString, "alpha", 2)[0])
			if err != nil {
				// fmt.Println("Error parsing major version:", err)
			}
			v.Major = ptr(maj)
		}

		// Check for stability level
		if strings.Contains(vString, "beta") || strings.Contains(vString, "alpha") {
			v.Stability = ptr("GA")
			if strings.Contains(vString, "beta") {
				v.Stability = ptr("beta")
			} else if strings.Contains(vString, "alpha") {
				v.Stability = ptr("alpha")
			}
			// Extract minor version and additional numbers
			additional := ""
			switch deptr(v.Stability) {
			case "Beta":
				additional = strings.TrimPrefix(strings.Split(vString, "beta")[1], "v")
			case "Alpha":
				additional = strings.TrimPrefix(strings.Split(vString, "alpha")[1], "v")
			default:
				additional = strings.TrimPrefix(vString, fmt.Sprintf("v%d", v.Major))
			}
			for _, numStr := range strings.Split(additional, "") {
				num, err := strconv.Atoi(numStr)
				if err == nil {
					v.Additional = ptr(num)
				}
			}
		}
	}

	return v
}

func (v *Version) Compare(other *Version) int {
	if deptr(v.Major) != deptr(other.Major) && v.Stability == nil && other.Stability == nil {
		return deptr(other.Major) - deptr(v.Major) // Descending order
	}
	if v.Stability == nil && other.Stability != nil {
		return 1 // GA versions come before Beta/Alpha
	}
	if v.Stability != nil && other.Stability == nil {
		return -1 // GA versions come before Beta/Alpha
	}
	if deptr(v.Stability) != deptr(other.Stability) {
		switch deptr(v.Stability) {
		case "beta":
			return -1 // Beta comes before Alpha
		case "alpha":
			return 1 // Alpha comes after Beta
		default:
			return 0 // Should not happen
		}
	}
	if deptr(v.Major) != deptr(other.Major) {
		return deptr(other.Major) - deptr(v.Major) // Descending order
	}
	if deptr(v.Additional) != deptr(other.Additional) {
		return deptr(other.Additional) - deptr(v.Additional) // Descending order
	}

	if len(v.OriginalStr) > 0 && len(other.OriginalStr) > 0 {
		return strings.Compare(v.OriginalStr, other.OriginalStr)
	}
	return 0 // Versions are equal
}

type VersionSlice []*Version

func (vs VersionSlice) Len() int {
	return len(vs)
}

func (vs VersionSlice) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs VersionSlice) Less(i, j int) bool {
	return vs[i].Compare(vs[j]) < 0
}

func SortVersions(versionStrings []string) []string {
	var stable []*Version
	var stability []*Version
	var unknown []*Version

	var versions []*Version
	for _, vStr := range versionStrings {
		version := NewVersion(vStr)
		if version.Stability == nil && version.Major != nil {
			stable = append(stable, version)
		} else if version.Stability != nil {
			stability = append(stability, version)
		} else {
			unknown = append(unknown, version)
		}
	}

	sort.Sort(VersionSlice(stable))
	sort.Sort(VersionSlice(stability))
	sort.Sort(VersionSlice(unknown))

	versions = append(versions, stable...)
	versions = append(versions, stability...)
	versions = append(versions, unknown...)

	versionStrings = []string{}
	for _, v := range versions {
		versionStrings = append(versionStrings, v.OriginalStr)
	}

	return versionStrings
}
