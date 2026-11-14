package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"github.com/google/uuid"
)

func importProfileFile(ctx context.Context, store db.DB, mbzc mbz.MusicBrainzCaller, filename string) error {
	switch {
	case strings.Contains(filename, "Streaming_History_Audio"):
		return importer.ImportSpotifyFile(ctx, store, mbzc, filename)
	case strings.Contains(filename, "maloja"):
		return importer.ImportMalojaFile(ctx, store, mbzc, filename)
	case strings.Contains(filename, "recenttracks"):
		return importer.ImportLastFMFile(ctx, store, mbzc, filename)
	case strings.Contains(filename, "listenbrainz"):
		return importer.ImportListenBrainzExport(ctx, store, mbzc, filename)
	// TuneTrail exports keep the Koito JSON schema; both names are accepted (ISSUE-111).
	case strings.Contains(filename, "tunetrail"), strings.Contains(filename, "koito"):
		return importer.ImportKoitoFile(ctx, store, filename)
	default:
		return fmt.Errorf("%w: unrecognized import format", importer.ErrInvalidFile)
	}
}
func profileImportHandler(store db.DB, mbzc mbz.MusicBrainzCaller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if contentType == "multipart/form-data" {
			uploadProfileImport(w, r, store, mbzc)
			return
		}
		body, err := utils.DecodeBody[struct {
			Filename string `json:"filename"`
		}](r)
		if err != nil || body.Filename == "" || path.Base(body.Filename) != body.Filename || strings.ContainsAny(body.Filename, `\\/`) || body.Filename == "." || body.Filename == ".." {
			utils.WriteError(w, "a simple filename is required", 400)
			return
		}
		filename := path.Join(fmt.Sprint(db.ProfileID(r.Context())), body.Filename)
		// Do not follow symlinks supplied in the import directory.
		dirInfo, dirErr := os.Lstat(path.Join(cfg.ConfigDir(), "import", fmt.Sprint(db.ProfileID(r.Context()))))
		if dirErr != nil || !dirInfo.IsDir() {
			utils.WriteError(w, "profile import directory not found", 404)
			return
		}
		info, err := os.Lstat(path.Join(cfg.ConfigDir(), "import", filename))
		if err != nil || !info.Mode().IsRegular() {
			utils.WriteError(w, "import file not found", 404)
			return
		}
		// Move a manually selected file out of the boot queue before processing.
		stageDir := path.Join(cfg.ConfigDir(), "import_staging")
		if err := os.MkdirAll(stageDir, 0700); err != nil {
			utils.WriteError(w, "could not prepare import", 500)
			return
		}
		stageInfo, err := os.Lstat(stageDir)
		if err != nil || !stageInfo.IsDir() {
			utils.WriteError(w, "could not prepare import", 500)
			return
		}
		stagePath := path.Join(stageDir, uuid.NewString()+"-manual")
		if err := os.Rename(path.Join(cfg.ConfigDir(), "import", filename), stagePath); err != nil {
			utils.WriteError(w, "could not prepare import", 500)
			return
		}
		if err := os.Chmod(stagePath, 0600); err != nil {
			logger.FromContext(r.Context()).Error().Err(err).Msg("Manual import remains in staging")
			utils.WriteError(w, "could not prepare import; file retained in staging", 500)
			return
		}
		ctx := importer.WithUploadSource(r.Context(), stagePath)
		if err := importProfileFile(ctx, store, mbzc, filename); err != nil {
			// Preserve manual input for recovery outside the boot queue.
			root := path.Join(cfg.ConfigDir(), "import_failed")
			failedDir := path.Join(root, fmt.Sprint(db.ProfileID(ctx)))
			quarantineErr := os.MkdirAll(root, 0700)
			if quarantineErr == nil {
				var info os.FileInfo
				info, quarantineErr = os.Lstat(root)
				if quarantineErr == nil && !info.IsDir() {
					quarantineErr = fmt.Errorf("invalid failed import directory")
				}
			}
			if quarantineErr == nil {
				quarantineErr = os.Mkdir(failedDir, 0700)
				if errors.Is(quarantineErr, os.ErrExist) {
					quarantineErr = nil
				}
			}
			if quarantineErr == nil {
				var info os.FileInfo
				info, quarantineErr = os.Lstat(failedDir)
				if quarantineErr == nil && !info.IsDir() {
					quarantineErr = fmt.Errorf("invalid profile failed import directory")
				}
			}
			if quarantineErr == nil {
				quarantineErr = os.Rename(stagePath, path.Join(failedDir, uuid.NewString()+"-"+body.Filename))
			}
			message := "import failed; some records may have been saved"
			if quarantineErr != nil {
				logger.FromContext(ctx).Error().Err(quarantineErr).Msg("Manual import remains in staging")
				message += "; file retained in staging"
			}
			status := http.StatusInternalServerError
			if errors.Is(err, importer.ErrInvalidFile) || errors.Is(err, db.ErrInvalidInput) {
				status = http.StatusBadRequest
			}
			utils.WriteError(w, message, status)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// Uploads have server-generated filenames; the selected format selects the
// existing parser, while the URL selects the destination independently of file content.
func uploadProfileImport(w http.ResponseWriter, r *http.Request, store db.DB, mbzc mbz.MusicBrainzCaller) {
	// Preserve the distinction between invalid uploads and infrastructure failures.
	r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		utils.WriteError(w, "import upload exceeds 100 MiB or is invalid", 400)
		return
	}
	defer r.MultipartForm.RemoveAll()
	formats := map[string]string{"tunetrail": "tunetrail.json", "koito": "koito.json", "listenbrainz": "listenbrainz.zip", "spotify": "Streaming_History_Audio.json", "lastfm": "recenttracks.json", "maloja": "maloja.json"}
	suffix, ok := formats[r.FormValue("format")]
	if !ok {
		utils.WriteError(w, "select a supported import format", 400)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		utils.WriteError(w, "import file required", 400)
		return
	}
	defer file.Close()
	// HTTP staging is outside import/, which the boot importer scans after a crash.
	dir := path.Join(cfg.ConfigDir(), "import_staging")
	if err = os.MkdirAll(dir, 0700); err != nil {
		utils.WriteError(w, "could not prepare import", 500)
		return
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		utils.WriteError(w, "invalid import staging directory", 500)
		return
	}
	out, err := os.CreateTemp(dir, "upload-*")
	if err != nil {
		utils.WriteError(w, "could not save import", 500)
		return
	}
	destination := out.Name()
	defer os.Remove(destination)
	_, err = io.Copy(out, file)
	closeErr := out.Close()
	if err != nil || closeErr != nil {
		utils.WriteError(w, "could not save import", 500)
		return
	}
	name := uuid.NewString() + "-" + suffix
	ctx := importer.WithUploadSource(r.Context(), destination)
	if err = importProfileFile(ctx, store, mbzc, path.Join(fmt.Sprint(db.ProfileID(ctx)), name)); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, importer.ErrInvalidFile) || errors.Is(err, db.ErrInvalidInput) {
			status = http.StatusBadRequest
		}
		utils.WriteError(w, "import failed; some records may have been saved. Check the file before retrying", status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
