package transcribe

import (
	"archive/tar"
	"compress/bzip2"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

// EnsureModel installs the Whisper model when it is missing. It is safe to
// call from several goroutines: one download runs and the others wait.
func (s *Service) EnsureModel(ctx context.Context, progress func(Progress)) error {
	s.modelMu.Lock()
	defer s.modelMu.Unlock()

	if modelInstalled(s.modelDir) {
		report(progress, "Speech model is ready.", 1)
		return nil
	}

	report(progress, "Preparing the speech model…", -1)
	if err := os.MkdirAll(s.modelsDir, 0o755); err != nil {
		return fmt.Errorf("create models directory: %w", err)
	}

	temporary, err := os.MkdirTemp(s.modelsDir, "whisper-download-*")
	if err != nil {
		return fmt.Errorf("create temporary model directory: %w", err)
	}
	defer os.RemoveAll(temporary)

	archive := filepath.Join(temporary, "model.tar.bz2")
	if err := downloadModel(ctx, s.modelURL, archive, progress); err != nil {
		return err
	}

	report(progress, "Extracting the speech model…", -1)
	extracted := filepath.Join(temporary, "extracted")
	if err := os.MkdirAll(extracted, 0o755); err != nil {
		return fmt.Errorf("create extraction directory: %w", err)
	}
	if err := extractWhisperModel(archive, extracted); err != nil {
		return err
	}
	if !modelFilesExist(extracted) {
		return errors.New("the downloaded model archive is incomplete")
	}

	if err := os.RemoveAll(s.modelDir); err != nil {
		return fmt.Errorf("remove incomplete model: %w", err)
	}
	if err := os.Rename(extracted, s.modelDir); err != nil {
		return fmt.Errorf("install speech model: %w", err)
	}
	if !modelInstalled(s.modelDir) {
		return errors.New("speech model installation could not be verified")
	}

	report(progress, "Speech model is ready.", 1)
	return nil
}

func modelInstalled(directory string) bool {
	return modelFilesExist(directory)
}

func modelFilesExist(directory string) bool {
	required := []string{
		"tiny.en-encoder.int8.onnx",
		"tiny.en-decoder.int8.onnx",
		"tiny.en-tokens.txt",
	}
	for _, name := range required {
		info, err := os.Stat(filepath.Join(directory, name))
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			return false
		}
	}
	return true
}

func downloadModel(ctx context.Context, url, destination string, progress func(Progress)) error {
	client := &http.Client{Timeout: 30 * time.Minute}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create model download request: %w", err)
	}
	request.Header.Set("User-Agent", "Amba-Read-Along")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download speech model: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download speech model: server returned HTTP %d", response.StatusCode)
	}

	file, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create model archive: %w", err)
	}
	defer file.Close()

	const bufferSize = 128 * 1024
	buffer := make([]byte, bufferSize)
	var downloaded, lastReported int64

	for {
		n, readErr := response.Body.Read(buffer)
		if n > 0 {
			written, writeErr := file.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("save model archive: %w", writeErr)
			}
			if written != n {
				return io.ErrShortWrite
			}
			downloaded += int64(written)

			if downloaded-lastReported >= 5*1024*1024 {
				lastReported = downloaded
				if response.ContentLength > 0 {
					report(progress, fmt.Sprintf(
						"Downloading the speech model: %.0f%% (%.1f MB of %.1f MB)",
						float64(downloaded)/float64(response.ContentLength)*100,
						float64(downloaded)/(1024*1024),
						float64(response.ContentLength)/(1024*1024),
					), float64(downloaded)/float64(response.ContentLength))
				} else {
					report(progress, fmt.Sprintf(
						"Downloading the speech model: %.1f MB",
						float64(downloaded)/(1024*1024),
					), -1)
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("download speech model: %w", readErr)
		}
	}

	if response.ContentLength >= 0 && downloaded != response.ContentLength {
		return fmt.Errorf(
			"incomplete model download: received %d bytes, expected %d",
			downloaded,
			response.ContentLength,
		)
	}
	if downloaded == 0 {
		return errors.New("the downloaded model archive is empty")
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("save model archive: %w", err)
	}
	return nil
}

func extractWhisperModel(archivePath, destination string) error {
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open model archive: %w", err)
	}
	defer archiveFile.Close()

	tarReader := tar.NewReader(bzip2.NewReader(archiveFile))
	required := map[string]bool{
		"tiny.en-encoder.int8.onnx": false,
		"tiny.en-decoder.int8.onnx": false,
		"tiny.en-tokens.txt":        false,
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read model archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}

		name := path.Clean(header.Name)
		if name == "." || path.IsAbs(name) {
			continue
		}
		filename := path.Base(name)
		if _, wanted := required[filename]; !wanted {
			continue
		}
		if header.Size <= 0 {
			return fmt.Errorf("invalid file size in model archive: %s", filename)
		}

		output, err := os.OpenFile(
			filepath.Join(destination, filename),
			os.O_CREATE|os.O_WRONLY|os.O_EXCL,
			0o644,
		)
		if err != nil {
			return fmt.Errorf("create extracted file %s: %w", filename, err)
		}
		_, copyErr := io.CopyN(output, tarReader, header.Size)
		closeErr := output.Close()
		if copyErr != nil {
			return fmt.Errorf("extract %s: %w", filename, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close extracted file %s: %w", filename, closeErr)
		}
		required[filename] = true
	}

	for filename, found := range required {
		if !found {
			return fmt.Errorf("model archive is missing %s", filename)
		}
	}
	return nil
}

func report(progress func(Progress), stage string, fraction float64) {
	if progress != nil {
		progress(Progress{Stage: stage, Fraction: fraction})
	}
}
