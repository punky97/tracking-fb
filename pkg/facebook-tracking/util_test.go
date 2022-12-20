package facebook_tracking

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseFacebookTracking(t *testing.T) {
	str := `pixel1/token1,pixel2/token2-testcode-testcode2,
	pixel3,,
	pixel4/token4`

	pixels := ParseFacebookTracking(str)

	assert.Equal(t, 4, len(pixels))

	assert.Equal(t, pixels[0].Id, "pixel1")
	assert.Equal(t, pixels[0].Token, "token1")
	assert.Equal(t, pixels[0].TestCode, "")

	assert.Equal(t, pixels[1].Id, "pixel2")
	assert.Equal(t, pixels[1].Token, "token2")
	assert.Equal(t, pixels[1].TestCode, "testcode2")

	assert.Equal(t, pixels[2].Id, "pixel3")
	assert.Equal(t, pixels[2].Token, "")
	assert.Equal(t, pixels[2].TestCode, "")

	assert.Equal(t, pixels[3].Id, "pixel4")
	assert.Equal(t, pixels[3].Token, "token4")
	assert.Equal(t, pixels[3].TestCode, "")
}

func TestParseFacebookTracking2(t *testing.T) {
	str := ``

	pixels := ParseFacebookTracking(str)

	assert.Equal(t, 0, len(pixels))


}
