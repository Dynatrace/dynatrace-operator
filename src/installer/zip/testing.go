package zip

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/src/installer/common"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TestZipDirName  = "test"
	TestZipFilename = "test.txt"

	TestRawZip  = `UEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAAIABwAdGVzdC50eHRVVAkAA3w0lWATB55gdXgLAAEE6AMAAAToAwAAeW91IGZvdW5kIHRoZSBlYXN0ZXIgZWdnClBLAwQKAAAAAADAOa5SAAAAAAAAAAAAAAAABQAcAHRlc3QvVVQJAAMXB55gHQeeYHV4CwABBOgDAAAE6AMAAFBLAwQKAAAAAACodKdSbC0hZxkAAAAZAAAADQAcAHRlc3QvdGVzdC50eHRVVAkAA3w0lWATB55gdXgLAAEE6AMAAAToAwAAeW91IGZvdW5kIHRoZSBlYXN0ZXIgZWdnClBLAwQKAAAAAADCOa5SAAAAAAAAAAAAAAAACgAcAHRlc3QvdGVzdC9VVAkAAxwHnmAgB55gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAASABwAdGVzdC90ZXN0L3Rlc3QudHh0VVQJAAN8NJVgHAeeYHV4CwABBOgDAAAE6AMAAHlvdSBmb3VuZCB0aGUgZWFzdGVyIGVnZwpQSwMECgAAAAAA2zquUgAAAAAAAAAAAAAAAAYAHABhZ2VudC9VVAkAAy4JnmAxCZ5gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAOI6rlIAAAAAAAAAAAAAAAALABwAYWdlbnQvY29uZi9VVAkAAzgJnmA+CZ5gdXgLAAEE6AMAAAToAwAAUEsDBAoAAAAAAKh0p1JsLSFnGQAAABkAAAATABwAYWdlbnQvY29uZi90ZXN0LnR4dFVUCQADfDSVYDgJnmB1eAsAAQToAwAABOgDAAB5b3UgZm91bmQgdGhlIGVhc3RlciBlZ2cKUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAAAgAGAAAAAAAAQAAAKSBAAAAAHRlc3QudHh0VVQFAAN8NJVgdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAwDmuUgAAAAAAAAAAAAAAAAUAGAAAAAAAAAAQAO1BWwAAAHRlc3QvVVQFAAMXB55gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAAA0AGAAAAAAAAQAAAKSBmgAAAHRlc3QvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwECHgMKAAAAAADCOa5SAAAAAAAAAAAAAAAACgAYAAAAAAAAABAA7UH6AAAAdGVzdC90ZXN0L1VUBQADHAeeYHV4CwABBOgDAAAE6AMAAFBLAQIeAwoAAAAAAKh0p1JsLSFnGQAAABkAAAASABgAAAAAAAEAAACkgT4BAAB0ZXN0L3Rlc3QvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwECHgMKAAAAAADbOq5SAAAAAAAAAAAAAAAABgAYAAAAAAAAABAA7UGjAQAAYWdlbnQvVVQFAAMuCZ5gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAA4jquUgAAAAAAAAAAAAAAAAsAGAAAAAAAAAAQAO1B4wEAAGFnZW50L2NvbmYvVVQFAAM4CZ5gdXgLAAEE6AMAAAToAwAAUEsBAh4DCgAAAAAAqHSnUmwtIWcZAAAAGQAAABMAGAAAAAAAAQAAAKSBKAIAAGFnZW50L2NvbmYvdGVzdC50eHRVVAUAA3w0lWB1eAsAAQToAwAABOgDAABQSwUGAAAAAAgACACKAgAAjgIAAAAA`
	TestRawGzip = `H4sIAAqxYmIAA+2cW2/iRhTHx2y2cbKqxEO3at9G2tfImfGMGVAVCchFibRtVyG77Wq3Sg04LFqwI9tssooi8b596rfpF+kH6SeoDUMw5AJ0GWdTn59kHS6+jIH//8zxjHlhn+87dtPxNxs933fcsNn20ZIhhAjLwkgMGEcioYRiyk1GLEY4NTExIyjC58tuyE30gtD2o6Z0bb/hdDpG4HxoBO3319bbef1T5eiwsr37dsfr2m0XvwwcP5g4yQh8FR8ItIhb7eYWJaLISpRzcz16pRe/whkpUmIRsW6WcDdsd50tWrBIgVmEFg2LsCI1LbF+3ycAfBbGpvpjjPTfjx6/3Pc+xbG898+fZMyk/qkwTYawpb5pmde/sfniKgMYx8ZO7bgWer6z1GNEn0eB87v837z6/pnJY/+nlgn+nwbz+D8jE/7PSwVu0FJRRColkAAeNoYy1Y8Z6b+PbvZ/KsiU/imN/Z8oas8EGdc/evztKsoh9KPdwD/X8K9YEr+G1qLFjJa9aImfDwx5tEY5f9suK0dHh/Lh+Wgr4MtkIv8r8oFZ+Z8xa1r/gluQ/9Pg8/N/0cLPD6qVw+39g1e7xrkdhr7R8LqGfXracYy9thv9tA7cE2/rYLvSGi7Vym7lTtajjFCLdvj89V07nOVE0DWZjaFM9WPuzv+UE8Kn9E8EEZD/U0Kr9poUoVhEOhpGTb95VV0u18glohbvo34WnNY7Xj16/tfSW6yGuO0rKEQOClCYbH/9tNMOQkL+1nKPVh5/taqv6k/032rvvLNaaIe9oGr7b+JnR57XqY8e2/VXbefsOP/NtueG0e/F8QcbtJtOtMrbXyIX886qXs9tBm8Sb+hr+tpx/vuLi0gM5gYuCnq5gS9KJtnAnBUuL9f0p882fzg8ft/put6nYbs1TZ7A11Mn9EfyhD7UfLfjuS00+IIAAAAAIEamBP3J/TYDAIAvkNgfsIxlGfvDqMn3czKuJLbJy4hlLMvYH0ZNrpeTcUVGXca8jFjGsoz9YZSmpcniQ5NHHhUvmrwsoGEZywudMgBkhkfDkI/z/+7t9T8AAP9jtJWd2k4V3X6NKM61OFp+H22AZDZH1zsBueHFwu/Q+H0sY1nG/jBCRwAAACBtkuP/oROEKo4xa/434fH4H+GCE8ZNMZj/I2D8PxXmGf+/7zYC6jAGqlc7CXy++d9J/UcOYMH87zRI+r/dclwVCWAB/+fcKsT+z6gA/08D8P9sYwxVrzQBLOD/Uv+EMQH+nwbT/X8jPF96Cpg1/3fk/5bJ4pm/8f0/hED/PxXA/7ONoUz1Y2bd/0MYndI/Gcz/h/m/6vno9fBJPA0Vh+8c7NhB6PjYabVA9Nlg1P8b9wIannuy3GPMqv8m7v+16PD/Hxjk/zT4T/f/FC2DsJKgtEgYGMWDZqT/WPWqisD56r+k/qlpEaj/0mDi+1dUCi5e/3HCCPh/GkD9l20m9K+oFFy8/jOFIFD/pQHUf9lGjv8qnQQyq/6bmP/B4P+f0gTyf7aR+lc6CWS++i+pf0qFCfVfGtzo/0vuAy5e/5miwMH/0wD8P9sk/F/ZIODi9V9k/xzqvzSA+i/bJPt/qjoBi+d/JjiM/6UC5P9sk9S/6vl/fTT/9V8C//+bCpD/AQAAssm/yo11zwBqAAA=`
)

func SetupInvalidTestZip(t *testing.T, fs afero.Fs) afero.File {
	zipFile := SetupTestArchive(t, fs, TestRawZip)

	_, err := zipFile.Seek(8, io.SeekStart)
	require.NoError(t, err)

	return zipFile
}

func SetupTestArchive(t *testing.T, fs afero.Fs, rawZip string) afero.File {
	zipf, err := base64.StdEncoding.DecodeString(rawZip)
	require.NoError(t, err)

	zipFile, err := afero.TempFile(fs, "", "")
	require.NoError(t, err)

	_, err = zipFile.Write(zipf)
	require.NoError(t, err)

	err = zipFile.Sync()
	require.NoError(t, err)

	_, err = zipFile.Seek(0, io.SeekStart)
	require.NoError(t, err)

	return zipFile
}

func testUnpackedArchive(t *testing.T, fs afero.Fs) {
	exists, err := afero.Exists(fs, filepath.Join(TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = afero.Exists(fs, filepath.Join(TestZipDirName, TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = afero.Exists(fs, filepath.Join(TestZipDirName, TestZipDirName, TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = afero.Exists(fs, filepath.Join(TestZipDirName, common.AgentConfDirPath, TestZipFilename))
	require.NoError(t, err)
	assert.True(t, exists)

	info, err := fs.Stat(filepath.Join(TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.False(t, info.IsDir())
	assert.Equal(t, int64(25), info.Size())

	info, err = fs.Stat(filepath.Join(TestZipDirName, TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.False(t, info.IsDir())
	assert.Equal(t, int64(25), info.Size())

	info, err = fs.Stat(filepath.Join(TestZipDirName, TestZipDirName, TestZipDirName, TestZipFilename))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.False(t, info.IsDir())
	assert.Equal(t, int64(25), info.Size())

	info, err = fs.Stat(filepath.Join(TestZipDirName, common.AgentConfDirPath, TestZipFilename))
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.False(t, info.IsDir())
	assert.Equal(t, int64(25), info.Size())

	mode := info.Mode().Perm() & 020
	// Assert file is group writeable
	assert.NotEqual(t, mode, os.FileMode(0))
}
