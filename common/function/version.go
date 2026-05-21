package function

import (
	"fmt"
	"regexp"
	"strconv"
)

func ExtractMajorVersion(version string) (int, error) {
	// 匹配版本号中的首个数字（主版本号）
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(version)
	if match == "" {
		return 0, fmt.Errorf("no digits found in version: %s", version)
	}

	major, err := strconv.Atoi(match)
	if err != nil {
		return 0, fmt.Errorf("invalid number format: %s", match)
	}
	return major, nil
}

// 处理整个版本号数组
func ExtractMajorVersions(versions []string) ([]int, error) {
	resultMap := make(map[int]struct{})

	for _, v := range versions {
		major, err := ExtractMajorVersion(v)
		if err != nil {
			return nil, err
		}
		resultMap[major] = struct{}{}
	}

	result := make([]int, 0, len(versions))
	for version, _ := range resultMap {
		result = append(result, version)
	}

	return result, nil
}
