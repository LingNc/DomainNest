package handler

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type ScriptDownloadHandler struct {
	scriptsDir string
}

func NewScriptDownloadHandler(scriptsDir string) *ScriptDownloadHandler {
	return &ScriptDownloadHandler{scriptsDir: scriptsDir}
}

func (h *ScriptDownloadHandler) Download1PanelPatch(c *gin.Context) {
	lang := c.Query("lang")
	if lang != "en-US" {
		lang = "zh-CN" // default to Chinese
	}

	readmeFile := fmt.Sprintf("README.%s.md", lang)
	patchFile := "1panel-v1-httpreq.patch"
	scriptFile := "patch-1panel-v1.sh"

	// Verify files exist
	for _, f := range []string{patchFile, scriptFile, readmeFile} {
		path := filepath.Join(h.scriptsDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "patch file not found: " + f})
			return
		}
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=\"1panel-v1-httpreq-patch.zip\"")

	zw := zip.NewWriter(c.Writer)
	defer zw.Close()

	// Add patch file
	if err := h.addFileToZip(zw, patchFile, patchFile); err != nil {
		return // headers already sent, can't return JSON
	}

	// Add install script
	if err := h.addFileToZip(zw, scriptFile, scriptFile); err != nil {
		return
	}

	// Add README with generic name
	if err := h.addFileToZip(zw, readmeFile, "README.md"); err != nil {
		return
	}
}

func (h *ScriptDownloadHandler) addFileToZip(zw *zip.Writer, srcName, dstName string) error {
	srcPath := filepath.Join(h.scriptsDir, srcName)
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = dstName
	header.Method = zip.Deflate

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, f)
	return err
}
