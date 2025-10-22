package File_Utils_M1

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
)

// ========== 日志开关：仅错误打印 ==========
// 置为 true 可恢复所有普通信息/警告打印；默认 false 静默普通信息与警告。
const verboseFS = false

// 统一输出函数（根据前缀与开关决定是否打印）
func logMsg(prefix, msg string) {
	// 只有错误（❌）一定打印；其它前缀（⚠️/🧹/✅）在 verboseFS=false 时静默
	if prefix == "❌" || verboseFS {
		fmt.Println(prefix, msg)
	}
}

// CreateDirectories 初始化目录结构
func CreateDirectories() {
	if M1_Public_Data.Dir == "" {
		logMsg("❌", "工作目录未设置，请先调用 SetWorkDir()")
		return
	}

	// 1️⃣ M1 基准目录
	base := filepath.Join(M1_Public_Data.Dir, "M1")

	// 如果 M1 已存在，保留它但清空 build 与 output
	if _, err := os.Stat(base); err == nil {
		buildPath := filepath.Join(base, "build")
		outputPath := filepath.Join(base, "output")

		for _, d := range []string{buildPath, outputPath} {
			if _, err := os.Stat(d); err == nil {
				logMsg("⚠️", fmt.Sprintf("检测到旧目录 %s，正在删除...", d))
				if err := os.RemoveAll(d); err != nil {
					logMsg("❌", fmt.Sprintf("删除旧目录失败 %s：%v", d, err))
					return
				}
				logMsg("🧹", fmt.Sprintf("已清理旧目录：%s", d))
			}
		}
	} else {
		// 如果 M1 目录不存在，则创建
		if err := os.MkdirAll(base, 0o755); err != nil {
			logMsg("❌", fmt.Sprintf("创建基准目录失败：%v", err))
			return
		}
	}

	// 2️⃣ 设置路径变量
	M1_Public_Data.BuildDir = filepath.Join(base, "build")
	M1_Public_Data.OutputDir = filepath.Join(base, "output")
	M1_Public_Data.LdiDir = filepath.Join(M1_Public_Data.OutputDir, "LDI")
	M1_Public_Data.TxtDir = filepath.Join(M1_Public_Data.OutputDir, "txt")

	// 3️⃣ 重新创建新目录结构
	for _, d := range []string{
		M1_Public_Data.BuildDir,
		M1_Public_Data.OutputDir,
		M1_Public_Data.LdiDir,
		M1_Public_Data.TxtDir,
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			logMsg("❌", fmt.Sprintf("创建目录失败 %s：%v", d, err))
			return
		}
	}

	// 4️⃣ 输出成功信息（静默，除非 verboseFS=true）
	logMsg("✅", "目录结构初始化成功："+base)
}

// CopyMatchingSLXFiles 复制符合规则的 slx 文件
func CopyMatchingSLXFiles(srcRoot string) {
	if M1_Public_Data.BuildDir == "" {
		logMsg("❌", "BuildDir 未设置，请先调用 CreateDirectories()")
		return
	}

	info, err := os.Stat(srcRoot)
	if err != nil {
		logMsg("❌", fmt.Sprintf("源路径不可用：%v", err))
		return
	}
	if !info.IsDir() {
		logMsg("❌", fmt.Sprintf("源路径不是目录：%s", srcRoot))
		return
	}

	entries, err := os.ReadDir(srcRoot)
	if err != nil {
		logMsg("❌", fmt.Sprintf("读取目录失败：%v", err))
		return
	}

	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		folderName := ent.Name()
		subdir := filepath.Join(srcRoot, folderName)
		expect := folderName + ".slx"

		matchPath, ok, err := findMatchingSLX(subdir, expect)
		if err != nil {
			logMsg("⚠️", fmt.Sprintf("子目录 %s 检索出错：%v", subdir, err))
			continue
		}
		if !ok {
			continue
		}

		dest := filepath.Join(M1_Public_Data.BuildDir, filepath.Base(matchPath))
		if err := copyFile(matchPath, dest); err != nil {
			logMsg("⚠️", fmt.Sprintf("复制失败 %s -> %s：%v", matchPath, dest, err))
			continue
		}
		// 成功信息静默，除非 verboseFS=true
		//logMsg("✅", fmt.Sprintf("已复制：%s", filepath.Base(matchPath)))
	}
}

// UnzipSLXsInBuildDir 解压 build 目录下所有 slx
func UnzipSLXsInBuildDir() {
	build := M1_Public_Data.BuildDir
	if build == "" {
		logMsg("❌", "BuildDir 未设置，请先调用 CreateDirectories()")
		return
	}

	ents, err := os.ReadDir(build)
	if err != nil {
		logMsg("❌", fmt.Sprintf("读取 BuildDir 失败：%v", err))
		return
	}

	count := 0
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.ToLower(filepath.Ext(name)) != ".slx" {
			continue
		}

		src := filepath.Join(build, name)
		base := strings.TrimSuffix(name, filepath.Ext(name))
		dest := filepath.Join(build, base)

		if _, err := os.Stat(dest); err == nil {
			_ = os.RemoveAll(dest)
		}
		_ = os.MkdirAll(dest, 0o755)

		n, err := unzipToDir(src, dest)
		if err != nil {
			logMsg("⚠️", fmt.Sprintf("解压失败 %s：%v", src, err))
			continue
		}
		// 成功信息静默，除非 verboseFS=true
		logMsg("✅", fmt.Sprintf("%s 解压完成（%d 文件）", name, n))
		count++
	}
	// 成功信息静默，除非 verboseFS=true
	logMsg("✅", fmt.Sprintf("已解压 %d 个模型", count))
}

// findMatchingSLX 在目录中查找匹配的 slx 文件
func findMatchingSLX(dir, expect string) (string, bool, error) {
	want := strings.ToLower(strings.TrimSpace(expect))
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", false, err
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.ToLower(filepath.Ext(name)) != ".slx" {
			continue
		}
		if strings.ToLower(name) == want {
			return filepath.Join(dir, name), true, nil
		}
	}
	return "", false, nil
}

// copyFile 覆盖式复制文件
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	df, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer df.Close()

	if _, err := io.Copy(df, sf); err != nil {
		return err
	}

	if si, err := os.Stat(src); err == nil {
		_ = os.Chmod(dst, si.Mode())
	}
	return nil
}

// unzipToDir 解压 slx 文件到目录
func unzipToDir(zipPath, destDir string) (int, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	count := 0
	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)
		cleaned := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleaned, destDir+string(os.PathSeparator)) && cleaned != destDir {
			return count, fmt.Errorf("zip entry 路径越界: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(cleaned, 0o755)
			continue
		}

		_ = os.MkdirAll(filepath.Dir(cleaned), 0o755)
		rc, err := f.Open()
		if err != nil {
			return count, err
		}
		out, err := os.OpenFile(cleaned, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return count, err
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return count, err
		}
		out.Close()
		rc.Close()
		count++
	}
	return count, nil
}
