package facebook_tracking

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
	"tracking-fb/dto"
)

func GetPixelByCode(code string) []*dto.Pixel {
	pixelWithToken := viper.GetString(fmt.Sprintf("tracking.%v", code))
	if len(pixelWithToken) < 1 {
		return nil
	}
	result := make([]*dto.Pixel, 0)
	pixel := dto.Pixel{}
	pixels := strings.Split(pixelWithToken, "/")
	if len(pixels) == 0 {
		return result
	}

	pixelId := strings.Trim(pixels[0], " \t")
	if len(pixelId) == 0 {
		return result
	}

	// Only pixel_id
	pixel.Id = pixelId
	if len(pixels) == 1 {
		result = append(result, &pixel)
		return result
	}

	// Pixel & token
	tokens := strings.Split(pixels[1], "-testcode-")

	// don't have test code
	pixel.Token = strings.Trim(tokens[0], " \t")
	if len(tokens) == 1 {
		result = append(result, &pixel)
		return result
	}

	//has test code
	pixel.TestCode = strings.Trim(tokens[1], " \t")
	result = append(result, &pixel)
	return result
}

func ParseFacebookTracking(raw string) []*dto.Pixel {
	lines := strings.Split(raw, "\n")

	result := make([]*dto.Pixel, 0)
	for _, line := range lines {
		pixelWithTokens := strings.Split(line, ",")
		for _, pixelWithToken := range pixelWithTokens {
			pixel := dto.Pixel{}
			pixels := strings.Split(pixelWithToken, "/")

			if len(pixels) == 0 {
				continue
			}

			pixelId := strings.Trim(pixels[0], " \t")
			if len(pixelId) == 0 {
				continue
			}

			// Only pixel_id
			pixel.Id = pixelId
			if len(pixels) == 1 {
				result = append(result, &pixel)
				continue
			}

			// Pixel & token
			tokens := strings.Split(pixels[1], "-testcode-")

			// don't have test code
			pixel.Token = strings.Trim(tokens[0], " \t")
			if len(tokens) == 1 {
				result = append(result, &pixel)
				continue
			}

			//has test code
			pixel.TestCode = strings.Trim(tokens[1], " \t")
			result = append(result, &pixel)
		}
	}

	return result
}
