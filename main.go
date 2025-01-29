package main

import (
	"archive/tar"
	"bytes"
	"cmp"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type PullRequest struct {
	// Base may be empty if we only want to evaluate current status.
	Base string `json:"base"`
	Head string `json:"head"`
}

type Response struct {
	SimpleFindings string `json:"simpleFindings"`
}

func internalError(w http.ResponseWriter, err string) {
	fmt.Printf("Internal error: %s\n", err)
	http.Error(w, fmt.Sprintf("Internal error: %s", err), http.StatusInternalServerError)
}

func handlePull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PullRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("Unable to decode body: %s\n", err)
		http.Error(w, fmt.Sprintf("Unable to read body: %s", err), http.StatusBadRequest)
		return
	}
	if req.Head == "" {
		http.Error(w, "head cannot be empty", http.StatusBadRequest)
		return
	}
	fmt.Printf("Attempting to analyze...\n")

	// Create temp directories
	tmpBase := cmp.Or(os.Getenv("TMPDIR"), "/tmp")
	tmpDir, err := os.MkdirTemp(tmpBase, "banditize")
	if err != nil {
		internalError(w, err.Error())
		return
	}
	defer os.RemoveAll(tmpDir)

	baseDir := filepath.Join(tmpDir, "base")
	headDir := filepath.Join(tmpDir, "head")
	if err := os.Mkdir(baseDir, 0700); err != nil {
		internalError(w, err.Error())
		return
	}
	if err := os.Mkdir(headDir, 0700); err != nil {
		internalError(w, err.Error())
		return
	}

	baselineFile := filepath.Join(tmpDir, "baseline.json")
	// Bandit crashes if you pass --baseline but the file doesn't exist
	os.WriteFile(baselineFile, []byte("{}"), 0644)

	if req.Base != "" {
		if err := unpackTarball(req.Base, baseDir); err != nil {
			internalError(w, err.Error())
			return
		}
		baseScan := exec.Command("bandit", "-f", "json", "-o", baselineFile, "-r", "." /*baseDir*/, "--exit-zero")
		baseScan.Dir = baseDir // Baselines only work in bandit if the filenames are the same
		if output, err := baseScan.CombinedOutput(); err != nil {
			fmt.Printf("Bandit output:\n%s\n", output)
			internalError(w, fmt.Sprintf("Baseline failed: %s", err.Error()))
			return
		}
	}

	if err := unpackTarball(req.Head, headDir); err != nil {
		internalError(w, err.Error())
		return
	}

	// Analyze with Bandit
	// TODO: add an ini file to ignore tests(?)
	headScan := exec.Command("bandit", "--baseline", baselineFile, "-o", "-", "-r", "." /*headDir*/)
	headScan.Dir = headDir // Baselines only work in bandit if the filenames are the same
	output, err := headScan.Output()
	if err == nil {
		// Bandit had no findings, (exit status 0), so we can return early
		resp := Response{
			SimpleFindings: "",
		}
		fmt.Printf("Success: no findings\n")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 1 {
		internalError(w, fmt.Sprintf("Bandit failed with exit code %d", exitErr.ExitCode()))
		return
	}

	// Prepare response
	resp := Response{
		SimpleFindings: string(output),
	}

	fmt.Printf("Success: findings\n")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func unpackTarball(encoded string, dir string) error {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(decoded)
	gzReader, err := gzip.NewReader(buf)
	if err != nil {
		return err
	}
	defer gzReader.Close()
	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if !filepath.IsLocal(header.Name) {
			return fmt.Errorf("tar contained invalid name %q", header.Name)
		}
		target := filepath.Join(dir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return err
			}
			file.Close()
		}
	}
	return nil
}

func main() {
	http.HandleFunc("/pull", handlePull)
	port := cmp.Or(os.Getenv("PORT"), "8080")
	log.Printf("Listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(":" + port, nil))
}
