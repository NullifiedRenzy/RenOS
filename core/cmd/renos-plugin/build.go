package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CmdBuild() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	manifest, err := LoadManifest(dir)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	if errs := manifest.Validate(); len(errs) > 0 {
		return fmt.Errorf("validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	fmt.Println("  Validated plugin.json")

	if err := buildGoCode(dir, manifest); err != nil {
		return err
	}

	distDir := filepath.Join(dir, "dist")
	if err := copyStaticAssets(dir, distDir, manifest); err != nil {
		return err
	}

	if err := createRPK(distDir, dir, manifest); err != nil {
		return err
	}

	fmt.Printf("  Build complete: %s.rpk\n", manifest.Name)
	return nil
}

func buildGoCode(dir string, m *PluginManifest) error {
	output := filepath.Join(dir, "dist", m.Name)
	fmt.Printf("  Compiling Go code -> %s\n", output)

	cmd := exec.Command("go", "build", "-o", output, ".")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}
	return nil
}

func copyStaticAssets(srcDir, distDir string, m *PluginManifest) error {
	entries := []string{
		"frontend",
		"worker",
		"schedules",
		"webhooks",
		"actions",
		"routes",
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry)
		if info, err := os.Stat(srcPath); err == nil && info.IsDir() {
			dstPath := filepath.Join(distDir, entry)
			if err := copyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("copying %s: %w", entry, err)
			}
			fmt.Printf("  Copied %s/\n", entry)
		}
	}

	manifestData, err := os.ReadFile(filepath.Join(srcDir, "plugin.json"))
	if err != nil {
		return fmt.Errorf("reading plugin.json: %w", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "plugin.json"), manifestData, 0644); err != nil {
		return fmt.Errorf("writing plugin.json to dist: %w", err)
	}

	return nil
}

func createRPK(distDir, srcDir string, m *PluginManifest) error {
	rpkPath := filepath.Join(srcDir, m.Name+".rpk")
	fmt.Printf("  Creating package: %s\n", rpkPath)

	f, err := os.Create(rpkPath)
	if err != nil {
		return fmt.Errorf("creating .rpk file: %w", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	err = filepath.WalkDir(distDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(distDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			_, err := w.Create(rel + "/")
			return err
		}
		fh, err := w.Create(rel)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(fh, src)
		return err
	})
	if err != nil {
		return fmt.Errorf("creating zip: %w", err)
	}

	return nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func CmdValidate() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	manifest, err := LoadManifest(dir)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	if errs := manifest.Validate(); len(errs) > 0 {
		fmt.Println("Validation FAILED:")
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("plugin.json has %d validation error(s)", len(errs))
	}

	fmt.Println("Validation PASSED")
	fmt.Printf("  Name:    %s\n", manifest.Name)
	fmt.Printf("  Version: %s\n", manifest.Version)
	fmt.Printf("  Author:  %s\n", manifest.Author)
	fmt.Printf("  Routes:  %d\n", len(manifest.Routes))
	fmt.Printf("  Workers: %d\n", len(manifest.Workers))
	fmt.Printf("  Schedules: %d\n", len(manifest.Schedules))
	fmt.Printf("  Webhooks: %d\n", len(manifest.Webhooks))
	fmt.Printf("  Sidebar Items: %d\n", len(manifest.Sidebar))
	fmt.Printf("  Pages: %d\n", len(manifest.Pages))
	return nil
}

func CmdPackage() error {
	if err := CmdBuild(); err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	manifest, err := LoadManifest(dir)
	if err != nil {
		return err
	}
	fmt.Printf("Package created: %s/%s.rpk\n", dir, manifest.Name)
	return nil
}

func CmdPublish() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	manifest, err := LoadManifest(dir)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	rpkPath := filepath.Join(dir, manifest.Name+".rpk")
	if _, err := os.Stat(rpkPath); os.IsNotExist(err) {
		return fmt.Errorf("package not found at %s; run 'renos-plugin package' first", rpkPath)
	}

	marketDir := filepath.Join(os.Getenv("HOME"), ".renos", "marketplace")
	if err := os.MkdirAll(marketDir, 0755); err != nil {
		return fmt.Errorf("creating marketplace directory: %w", err)
	}

	dst := filepath.Join(marketDir, manifest.Name+".rpk")
	data, err := os.ReadFile(rpkPath)
	if err != nil {
		return fmt.Errorf("reading package: %w", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("publishing package: %w", err)
	}

	fmt.Printf("Published %s.rpk to marketplace (%s)\n", manifest.Name, marketDir)
	return nil
}

func CmdInstall(pluginPath string) error {
	installDir := filepath.Join(os.Getenv("HOME"), ".renos", "plugins")

	rpkPath := pluginPath
	info, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("plugin path not found: %w", err)
	}

	if info.IsDir() {
		manifest, err := LoadManifest(pluginPath)
		if err != nil {
			return fmt.Errorf("invalid plugin directory: %w", err)
		}
		target := filepath.Join(installDir, manifest.Name)
		if err := copyDir(pluginPath, target); err != nil {
			return fmt.Errorf("installing plugin: %w", err)
		}
		fmt.Printf("Installed %s to %s\n", manifest.Name, target)
		return nil
	}

	if filepath.Ext(pluginPath) == ".rpk" {
		targetDir := filepath.Join(installDir, strings.TrimSuffix(filepath.Base(rpkPath), ".rpk"))
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
		if err := extractZip(rpkPath, targetDir); err != nil {
			return fmt.Errorf("extracting package: %w", err)
		}
		fmt.Printf("Installed %s to %s\n", filepath.Base(rpkPath), targetDir)
		return nil
	}

	return fmt.Errorf("unsupported plugin path: %s (expected .rpk or plugin directory)", pluginPath)
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid zip path: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0755)
		src, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.Create(target)
		if err != nil {
			src.Close()
			return err
		}
		_, err = io.Copy(dst, src)
		src.Close()
		dst.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func CmdTest() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	fmt.Println("Running plugin tests...")
	cmd := exec.Command("go", "test", "./...", "-v")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}
	return nil
}
