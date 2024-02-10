package cmdline

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplace(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	c := testingCmdline("foo bar")

	require.NoError(c.Replace("baz"))
	assert.Equal("baz    ", string(c.handle.(*testingCmdlineHandle).content))

	require.NoError(c.Replace(""))
	assert.Equal("       ", string(c.handle.(*testingCmdlineHandle).content))

	require.Error(c.Replace("1234567890123456789012345678901234567890123456789012345678901234567890"))
	assert.Equal("       ", string(c.handle.(*testingCmdlineHandle).content))
}

func TestAppend(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// full length append
	c := testingCmdline("                   ")
	require.NoError(c.Append("123456789abcdefghij"))
	assert.Equal("123456789abcdefghij", string(c.handle.(*testingCmdlineHandle).content))

	// overfull length append
	c = testingCmdline("                   ")
	require.Error(c.Append("123456789abcdefghijk"))
	assert.Equal("                   ", string(c.handle.(*testingCmdlineHandle).content))

	// iterative append
	c = testingCmdline("                   ")
	require.NoError(c.Append("foo"))
	assert.Equal("foo                ", string(c.handle.(*testingCmdlineHandle).content))
	require.NoError(c.Append("bar"))
	assert.Equal("foo bar            ", string(c.handle.(*testingCmdlineHandle).content))
	require.NoError(c.Append("baz"))
	assert.Equal("foo bar baz        ", string(c.handle.(*testingCmdlineHandle).content))
	require.NoError(c.Append("1234567"))
	assert.Equal("foo bar baz 1234567", string(c.handle.(*testingCmdlineHandle).content))
	// now it's full
	require.Error(c.Append("8"))
	assert.Equal("foo bar baz 1234567", string(c.handle.(*testingCmdlineHandle).content))
}

func TestSet(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// keys without values
	c := testingCmdline("foo bar")
	require.NoError(c.Set(map[string]string{
		"baz": "",
		"qux": "",
	}, false))
	assert.Equal("baz qux", string(c.handle.(*testingCmdlineHandle).content))

	// keys with values
	c = testingCmdline("foo=1 bar=2")
	require.NoError(c.Set(map[string]string{
		"baz": "3",
		"qux": "4",
	}, false))
	assert.Equal("baz=3 qux=4", string(c.handle.(*testingCmdlineHandle).content))

	// keys with values and keep existing and overwriting
	c = testingCmdline("foo=1 bar=2            ")
	require.NoError(c.Set(map[string]string{
		"foo": "4", // overwrite foo but keep bar
		"baz": "3",
		"qux": "5",
	}, true))
	assert.Equal("bar=2 baz=3 foo=4 qux=5", string(c.handle.(*testingCmdlineHandle).content))

	// not enough space
	c = testingCmdline("foo=1 bar=2     ")
	require.Error(c.Set(map[string]string{
		"baz": "3",
	}, true))
	assert.Equal("foo=1 bar=2     ", string(c.handle.(*testingCmdlineHandle).content))
}

func TestSetOne(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// key without value rewrite
	c := testingCmdline("foo bar    ")
	require.NoError(c.SetOne("baz", "", false))
	assert.Equal("bar baz foo", string(c.handle.(*testingCmdlineHandle).content))

	// key with smaller value rewrite
	c = testingCmdline("mykey=originalvalue bar=2")
	require.NoError(c.SetOne("mykey", "newvalue", true))
	assert.Equal("mykey=newvalue      bar=2", string(c.handle.(*testingCmdlineHandle).content))

	// key with exact size value rewrite
	c = testingCmdline("a mykey=original b")
	require.NoError(c.SetOne("mykey", "newvalue", true))
	assert.Equal("a mykey=newvalue b", string(c.handle.(*testingCmdlineHandle).content))

	// key with larger value rewrite
	c = testingCmdline("a mykey=original b")
	require.Error(c.SetOne("mykey", "newvalue1234567890", true))
	assert.Equal("a mykey=original b", string(c.handle.(*testingCmdlineHandle).content))

	// key missing rewrite
	c = testingCmdline("someotherkey=original")
	require.Error(c.SetOne("key", "value", true))
	assert.Equal("someotherkey=original", string(c.handle.(*testingCmdlineHandle).content))
}

func testingCmdline(content string) *Cmdline {
	return New(&testingCmdlineHandle{
		content: []byte(content),
	}, int64(len(content)))
}

type testingCmdlineHandle struct {
	content []byte
}

func (h *testingCmdlineHandle) WriteAt(p []byte, off int64) (n int, err error) {
	n = copy(h.content[off:], p)
	if n < len(p) {
		err = io.EOF
	}
	return
}

func (h *testingCmdlineHandle) ReadAt(p []byte, off int64) (n int, err error) {
	n = copy(p, h.content[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}
