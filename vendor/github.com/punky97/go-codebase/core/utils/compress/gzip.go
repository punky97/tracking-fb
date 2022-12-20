package compress

import (
	"archive/tar"
	"bytes"
	"github.com/punky97/go-codebase/core/logger"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func GzipTarCompress(files []string, dest *os.File) error {
	wrt := gzip.NewWriter(dest)
	defer wrt.Close()

	tarWrt := tar.NewWriter(wrt)
	defer tarWrt.Close()

	for _, filePath := range files {
		rFile, err := os.Open(filePath)
		if err != nil {
			logger.BkLog.Errorw(fmt.Sprintf("Error when open: %v", err), "path", filePath)
			return err
		}

		defer rFile.Close()

		header := new(tar.Header)
		header.Name = filepath.Base(filePath)
		if stat, sterr := rFile.Stat(); sterr == nil {
			header.Size = stat.Size()
			header.Mode = int64(stat.Mode())
			header.ModTime = stat.ModTime()

			err = tarWrt.WriteHeader(header)
			if err != nil {
				logger.BkLog.Errorw(fmt.Sprintf("Error when write header: %v", err))
			}

			_, err = io.Copy(tarWrt, rFile)
			if err != nil {
				logger.BkLog.Errorw(fmt.Sprintf("Error when read file: %v", err), "path", filePath)
				return err
			}
		} else {
			logger.BkLog.Errorw(fmt.Sprintf("Error when get stat of file: %v", err), "path", filePath)
		}
	}

	return nil
}

func GUnzipData(data []byte) (resData []byte, err error) {
	b := bytes.NewBuffer(data)

	r, err := gzip.NewReader(b)
	if err != nil {
		return
	}

	defer r.Close()
	var resB bytes.Buffer
	_, err = resB.ReadFrom(r)
	if err != nil {
		return
	}

	resData = resB.Bytes()
	return
}

func GZipData(data []byte) (compressedData []byte, err error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)

	_, err = gz.Write(data)
	if err != nil {
		return
	}

	if err = gz.Flush(); err != nil {
		return
	}

	if err = gz.Close(); err != nil {
		return
	}

	compressedData = b.Bytes()

	return
}
