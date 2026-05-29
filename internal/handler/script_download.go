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
	if !filepath.IsAbs(scriptsDir) {
		if exe, err := os.Executable(); err == nil {
			scriptsDir = filepath.Join(filepath.Dir(exe), scriptsDir)
		}
	}
	return &ScriptDownloadHandler{scriptsDir: scriptsDir}
}

func (h *ScriptDownloadHandler) Download1PanelPatch(c *gin.Context) {
	lang := c.Query("lang")
	if lang != "en-US" {
		lang = "zh-CN" // default to Chinese
	}

	readmeFile := fmt.Sprintf("README.%s.md", lang)
	scriptFile := fmt.Sprintf("patch-1panel.%s.sh", lang)
	unpatchFile := fmt.Sprintf("unpatch-1panel.%s.sh", lang)
	patchV1 := "1panel-v1-httpreq.patch"
	patchV2 := "1panel-v2-httpreq.patch"
	patchFE := "1panel-httpreq-frontend.patch"

	// Verify files exist
	for _, f := range []string{patchV1, patchV2, patchFE, scriptFile, unpatchFile, readmeFile} {
		path := filepath.Join(h.scriptsDir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "patch file not found: " + f})
			return
		}
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=\"1panel-httpreq-patch.zip\"")

	zw := zip.NewWriter(c.Writer)
	defer zw.Close()

	// Add both patch files
	if err := h.addFileToZip(zw, patchV1, patchV1); err != nil {
		return
	}
	if err := h.addFileToZip(zw, patchV2, patchV2); err != nil {
		return
	}
	// Add frontend patch
	if err := h.addFileToZip(zw, patchFE, patchFE); err != nil {
		return
	}

	// Add install script
	if err := h.addFileToZip(zw, scriptFile, "patch-1panel.sh"); err != nil {
		return
	}

	// Add uninstall script
	if err := h.addFileToZip(zw, unpatchFile, "unpatch-1panel.sh"); err != nil {
		return
	}

	// Add README
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
